package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/checkins"
	"github.com/sidnevart/proof-forge/backend/internal/circles"
	"github.com/sidnevart/proof-forge/backend/internal/goals"
	"github.com/sidnevart/proof-forge/backend/internal/inspiration"
	"github.com/sidnevart/proof-forge/backend/internal/notifications"
	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/internal/recaps"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/callbacks"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/commands"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// ─── mock Telegram sender ────────────────────────────────────────────────────

type sentMessage struct {
	chatID    int64
	text      string
	messageID int // set by SendMessageWithKeyboard
	keyboard  *bot.InlineKeyboardMarkup
}

type mockTelegramSender struct {
	mu       sync.Mutex
	messages []sentMessage
}

func (m *mockTelegramSender) SendMessage(_ context.Context, chatID int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, sentMessage{chatID: chatID, text: text})
	return nil
}

func (m *mockTelegramSender) SendMessageWithKeyboard(_ context.Context, chatID int64, text string, kb bot.InlineKeyboardMarkup) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	kbCopy := kb
	m.messages = append(m.messages, sentMessage{chatID: chatID, text: text, keyboard: &kbCopy, messageID: 42})
	return 42, nil
}

func (m *mockTelegramSender) EditMessageText(_ context.Context, chatID int64, _ int, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, sentMessage{chatID: chatID, text: text})
	return nil
}

func (m *mockTelegramSender) AnswerCallbackQuery(_ context.Context, _, _ string) error { return nil }

// LastMessage returns the most recently sent message, or nil.
func (m *mockTelegramSender) LastMessage() *sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return nil
	}
	msg := m.messages[len(m.messages)-1]
	return &msg
}

// FindKeyboardMessage returns the first captured message that carries an inline keyboard.
func (m *mockTelegramSender) FindKeyboardMessage() *sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages {
		if m.messages[i].keyboard != nil {
			msg := m.messages[i]
			return &msg
		}
	}
	return nil
}

// FindMessagesForChat returns all messages sent to a specific chatID.
func (m *mockTelegramSender) FindMessagesForChat(chatID int64) []sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []sentMessage
	for _, msg := range m.messages {
		if msg.chatID == chatID {
			out = append(out, msg)
		}
	}
	return out
}

func (m *mockTelegramSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

// ─── mock Email sender ────────────────────────────────────────────────────────

type mockEmailSender struct {
	mu             sync.Mutex
	invites        []email.BuddyInviteParams
	acceptedEmails []email.BuddyAcceptedParams
}

func (m *mockEmailSender) SendBuddyInvite(_ context.Context, p email.BuddyInviteParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invites = append(m.invites, p)
	return nil
}

func (m *mockEmailSender) SendBuddyAccepted(_ context.Context, p email.BuddyAcceptedParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acceptedEmails = append(m.acceptedEmails, p)
	return nil
}

func (m *mockEmailSender) LastBuddyAccepted() *email.BuddyAcceptedParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.acceptedEmails) == 0 {
		return nil
	}
	p := m.acceptedEmails[len(m.acceptedEmails)-1]
	return &p
}

func (m *mockEmailSender) LastBuddyInvite() *email.BuddyInviteParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.invites) == 0 {
		return nil
	}
	p := m.invites[len(m.invites)-1]
	return &p
}

func (m *mockEmailSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invites = nil
	m.acceptedEmails = nil
}

// ─── seed helpers ─────────────────────────────────────────────────────────────

// linkTelegram directly inserts a telegram_links row, simulating /start link_<token>.
func linkTelegram(t *testing.T, pool *pgxpool.Pool, userID, chatID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO telegram_links (user_id, telegram_chat_id, status, linked_at)
		 VALUES ($1, $2, 'active', $3)
		 ON CONFLICT (user_id) DO UPDATE SET telegram_chat_id = EXCLUDED.telegram_chat_id, status = 'active'`,
		userID, chatID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("linkTelegram: %v", err)
	}
}

// getUserID looks up a user's ID by email (for linking Telegram after registration).
func getUserIDByEmail(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err != nil {
		t.Fatalf("getUserIDByEmail(%q): %v", email, err)
	}
	return id
}

// countDomainEvents returns the count of unprocessed domain_events of a given kind.
func countUnprocessedEvents(t *testing.T, pool *pgxpool.Pool, kind string) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM domain_events WHERE kind = $1 AND processed_at IS NULL`, kind,
	).Scan(&count)
	if err != nil {
		t.Fatalf("countUnprocessedEvents(%q): %v", kind, err)
	}
	return count
}

// ─── test router builder ──────────────────────────────────────────────────────

// testConfig builds a minimal platformconfig.Config for integration tests.
func testConfig() platformconfig.Config {
	return platformconfig.Config{
		App: platformconfig.AppConfig{
			Name:      "proofforge-test",
			Env:       "test",
			WebOrigin: "http://localhost:3000",
		},
		DB: platformconfig.DBConfig{
			HealthcheckTimeout: 2 * time.Second,
		},
		Invite: platformconfig.InviteConfig{
			TTL: 7 * 24 * time.Hour,
		},
		Session: platformconfig.SessionConfig{
			CookieName: "pf_session",
			TTL:        24 * time.Hour,
		},
	}
}

// buildNotifRouter wires the full API router with injected mock email and
// Telegram senders, and returns the router plus a NudgeEngine whose Tick()
// can be called synchronously in tests.
func buildNotifRouter(
	t *testing.T,
	pool *pgxpool.Pool,
	tgSender bot.Sender,
	emailSender email.Sender,
) (*chi.Mux, *notifications.NudgeEngine) {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := testConfig()

	notifRepo := notifications.NewPostgresRepository(pool)
	dispatcher := notifications.NewDispatcher(tgSender, notifRepo, log)
	engine := notifications.NewNudgeEngine(notifRepo, dispatcher, log)

	usersRepo := users.NewPostgresRepository(pool)
	usersSvc := users.NewService(usersRepo, usersRepo, cfg.Session.TTL)
	usersHandler := users.NewHandler(log, usersSvc, cfg.Session.CookieName, false)

	circlesService := circles.NewService(circles.NewPostgresRepository(pool))
	goalsHandler := goals.NewHandler(
		log,
		goals.NewService(
			goals.NewPostgresRepository(pool),
			emailSender,
			cfg.App.WebOrigin,
			log,
			cfg.Invite.TTL,
			goals.WithNotifEmitter(notifRepo),
			goals.WithCircleLister(circlesService),
		),
		analytics.NoopRecorder{},
	)

	circlesHandler := circles.NewHandler(circlesService)

	checkinsSvc := checkins.NewService(
		checkins.NewPostgresRepository(pool),
		checkins.NoopStorage{},
		notifRepo,
	)
	checkinsHandler := checkins.NewHandler(log, checkinsSvc)

	recapsHandler := recaps.NewHandler(
		log,
		recaps.NewService(recaps.NewPostgresRepository(pool), recaps.NoopProvider{}, log),
	)

	tgRepo := telegram.NewRepository(pool)
	reviewCb := callbacks.NewReviewHandler(checkinsSvc, tgRepo, tgSender, log)
	startCmd := commands.NewStartHandler(tgRepo, tgSender, log)
	cmdHandler := commands.NewCommandHandler(tgRepo, notifRepo, tgSender, cfg.App.WebOrigin, log)

	telegramHandler := telegram.NewHandler(telegram.HandlerConfig{
		Secret:         "", // no secret in tests — simpler
		StartHandler:   startCmd,
		CommandHandler: cmdHandler,
		ReviewCallback: reviewCb,
		Log:            log,
	})

	inspRepo := inspiration.NewPostgresRepository(pool)
	inspSvc := inspiration.NewService(inspRepo)
	inspHandler := inspiration.NewHandler(log, inspSvc)

	router := httpx.NewRouter(log, readiness.NewService(pool, 2*time.Second), cfg.App.WebOrigin)
	telegramHandler.RegisterRoutes(router)

	router.Route("/v1", func(r chi.Router) {
		usersHandler.RegisterPublicRoutes(r)
		goalsHandler.RegisterPublicRoutes(r)
		inspHandler.RegisterPublicRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(usersHandler.AuthMiddleware)
			usersHandler.RegisterProtectedRoutes(r)
			circlesHandler.RegisterRoutes(r)
			goalsHandler.RegisterRoutes(r)
			checkinsHandler.RegisterRoutes(r)
			recapsHandler.RegisterRoutes(r)
			inspHandler.RegisterRoutes(r)
		})
	})

	return router, engine
}
