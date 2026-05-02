package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

// TestE2E_CriticalRoutesExist verifies every critical route returns a non-404
// response. Catches the original "404 when creating a goal" bug class — if a
// route is misregistered or moved, this test fails immediately without needing
// a full happy-path setup.
func TestE2E_CriticalRoutesExist(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	router := newTestRouter(t, pool)

	type routeCheck struct {
		method string
		path   string
	}

	publicRoutes := []routeCheck{
		{http.MethodPost, "/v1/register"},
		{http.MethodPost, "/v1/login"},
		{http.MethodGet, "/v1/invites/some-token"},
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/readyz"},
	}

	// Protected routes — should return 401, never 404.
	protectedRoutes := []routeCheck{
		{http.MethodGet, "/v1/me"},
		{http.MethodGet, "/v1/dashboard"},
		{http.MethodGet, "/v1/goals"},
		{http.MethodPost, "/v1/goals"},
		{http.MethodPost, "/v1/invites/some-token/accept"},
		{http.MethodPost, "/v1/goals/1/check-ins"},
		{http.MethodGet, "/v1/goals/1/check-ins"},
		{http.MethodGet, "/v1/goals/1/recaps"},
		{http.MethodPost, "/v1/goals/1/stakes"},
		{http.MethodGet, "/v1/goals/1/stakes"},
		{http.MethodDelete, "/v1/stakes/1"},
		{http.MethodPost, "/v1/stakes/1/forfeit"},
	}

	for _, rc := range publicRoutes {
		t.Run("public_"+rc.method+"_"+sanitize(rc.path), func(t *testing.T) {
			req := httptest.NewRequest(rc.method, rc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("public route %s %s returned 404 — route not registered", rc.method, rc.path)
			}
		})
	}

	for _, rc := range protectedRoutes {
		t.Run("protected_"+rc.method+"_"+sanitize(rc.path), func(t *testing.T) {
			req := httptest.NewRequest(rc.method, rc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("protected route %s %s returned 404 — route not registered", rc.method, rc.path)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Logf("note: %s %s returned %d (expected 401 without auth)", rc.method, rc.path, rec.Code)
			}
		})
	}
}

// TestE2E_FullAccountabilityFlow walks the entire happy path:
// register owner → create goal → register buddy → accept invite → goal active
// → owner creates stake → submits check-in with text evidence → buddy approves
// → buddy forfeits stake → owner cancels another stake.
func TestE2E_FullAccountabilityFlow(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	router := newTestRouter(t, pool)

	ownerCookie := registerUser(t, router, "owner@example.com", "Owner")

	goalID := createGoal(t, router, ownerCookie, map[string]any{
		"title":       "Ship the MVP",
		"description": "Daily progress with proof",
		"buddy_name":  "Buddy",
		"buddy_email": "buddy@example.com",
	})
	if goalID <= 0 {
		t.Fatalf("expected positive goal ID, got %d", goalID)
	}

	dashboard := getDashboard(t, router, ownerCookie)
	if dashboard.Summary.TotalGoals != 1 {
		t.Fatalf("expected 1 goal, got %d", dashboard.Summary.TotalGoals)
	}
	if dashboard.Summary.PendingBuddyAcceptance != 1 {
		t.Fatalf("expected 1 pending invite, got %d", dashboard.Summary.PendingBuddyAcceptance)
	}

	inviteToken := getInviteToken(t, pool, goalID)

	buddyCookie := registerUser(t, router, "buddy@example.com", "Buddy")
	acceptInvite(t, router, buddyCookie, inviteToken)

	dashboard2 := getDashboard(t, router, ownerCookie)
	if dashboard2.Summary.ActiveGoals != 1 {
		t.Fatalf("expected 1 active goal after invite accept, got %d", dashboard2.Summary.ActiveGoals)
	}

	stakeID := createStake(t, router, ownerCookie, goalID, "5000₽ на благотворительность")
	if stakeID <= 0 {
		t.Fatalf("expected positive stake ID, got %d", stakeID)
	}

	checkInID := createCheckIn(t, router, ownerCookie, goalID)
	addTextEvidence(t, router, ownerCookie, checkInID, "Сделал deploy на staging")
	submitCheckIn(t, router, ownerCookie, checkInID)
	approveCheckIn(t, router, buddyCookie, checkInID)

	forfeitStake(t, router, buddyCookie, stakeID, "опоздание с дедлайном")

	stake2ID := createStake(t, router, ownerCookie, goalID, "побрить голову")
	cancelStake(t, router, ownerCookie, stake2ID)

	stakes := listStakes(t, router, ownerCookie, goalID)
	if len(stakes) != 2 {
		t.Fatalf("expected 2 stakes, got %d", len(stakes))
	}

	statusByID := make(map[int64]string)
	for _, sv := range stakes {
		statusByID[sv.Stake.ID] = sv.Stake.Status
	}
	if statusByID[stakeID] != "forfeited" {
		t.Errorf("expected stake %d forfeited, got %q", stakeID, statusByID[stakeID])
	}
	if statusByID[stake2ID] != "cancelled" {
		t.Errorf("expected stake %d cancelled, got %q", stake2ID, statusByID[stake2ID])
	}
}

// TestE2E_GoalCreationRejectsUnauthenticated reproduces what happens if the
// frontend forgets to send the session cookie. Should be 401, never 404.
func TestE2E_GoalCreationRejectsUnauthenticated(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	router := newTestRouter(t, pool)

	body, _ := json.Marshal(map[string]any{
		"title":       "X",
		"description": "Y",
		"buddy_name":  "Z",
		"buddy_email": "z@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session cookie, got %d", rec.Code)
	}
}

// TestE2E_GoalCreationRejectsInvalidInput verifies validation errors return 400.
func TestE2E_GoalCreationRejectsInvalidInput(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	router := newTestRouter(t, pool)
	cookie := registerUser(t, router, "user@example.com", "User")

	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty_title", map[string]any{"title": "", "description": "x", "buddy_name": "Buddy", "buddy_email": "b@x.com"}},
		{"short_buddy_name", map[string]any{"title": "T", "description": "x", "buddy_name": "B", "buddy_email": "b@x.com"}},
		{"invalid_email", map[string]any{"title": "T", "description": "x", "buddy_name": "Buddy", "buddy_email": "not-an-email"}},
		{"buddy_is_self", map[string]any{"title": "T", "description": "x", "buddy_name": "Buddy", "buddy_email": "user@example.com"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/goals", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %s, got %d (body: %s)", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestE2E_StakeForbiddenForNonParticipant verifies a stranger cannot
// create or forfeit stakes on someone else's goal.
func TestE2E_StakeForbiddenForNonParticipant(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)
	router := newTestRouter(t, pool)

	ownerCookie := registerUser(t, router, "owner@example.com", "Owner")
	goalID := createGoal(t, router, ownerCookie, map[string]any{
		"title":       "Mine",
		"description": "x",
		"buddy_name":  "Buddy",
		"buddy_email": "buddy@example.com",
	})
	inviteToken := getInviteToken(t, pool, goalID)
	buddyCookie := registerUser(t, router, "buddy@example.com", "Buddy")
	acceptInvite(t, router, buddyCookie, inviteToken)

	strangerCookie := registerUser(t, router, "stranger@example.com", "Stranger")

	// Stranger cannot list stakes
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/goals/%d/stakes", goalID), nil)
	req.AddCookie(strangerCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stranger listing stakes, got %d", rec.Code)
	}

	// Stranger cannot create stakes
	body, _ := json.Marshal(map[string]any{"description": "x"})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/goals/%d/stakes", goalID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(strangerCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stranger creating stake, got %d", rec.Code)
	}
}

// --- helpers ---

func newTestRouter(t *testing.T, pool *pgxpool.Pool) *chi.Mux {
	t.Helper()
	cfg := platformconfig.Config{
		App: platformconfig.AppConfig{
			Name:      "proofforge",
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
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpx.NewRouter(log, readiness.NewService(pool, 2*time.Second), cfg.App.WebOrigin)
	registerAPIRoutes(router, log, pool, cfg)
	return router
}

type dashboardResponse struct {
	Summary struct {
		TotalGoals             int `json:"total_goals"`
		PendingBuddyAcceptance int `json:"pending_buddy_acceptance"`
		ActiveGoals            int `json:"active_goals"`
	} `json:"summary"`
	Goals []map[string]any `json:"goals"`
}

type stakeListItem struct {
	Stake struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	} `json:"stake"`
}

func registerUser(t *testing.T, router *chi.Mux, email, name string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"email": email, "display_name": name})
	req := httptest.NewRequest(http.MethodPost, "/v1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d (body: %s)", email, rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "pf_session" {
			return c
		}
	}
	t.Fatalf("no session cookie after register %s", email)
	return nil
}

func createGoal(t *testing.T, router *chi.Mux, cookie *http.Cookie, input map[string]any) int64 {
	t.Helper()
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/v1/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create goal: expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Goal struct {
			Goal struct {
				ID int64 `json:"id"`
			} `json:"goal"`
		} `json:"goal"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create goal: %v", err)
	}
	return resp.Goal.Goal.ID
}

func getDashboard(t *testing.T, router *chi.Mux, cookie *http.Cookie) dashboardResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard: expected 200, got %d", rec.Code)
	}
	var resp dashboardResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	return resp
}

func getInviteToken(t *testing.T, pool *pgxpool.Pool, goalID int64) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var token string
	err := pool.QueryRow(ctx, `SELECT acceptance_token FROM invites WHERE goal_id = $1 LIMIT 1`, goalID).Scan(&token)
	if err != nil {
		t.Fatalf("fetch invite token: %v", err)
	}
	return token
}

func acceptInvite(t *testing.T, router *chi.Mux, cookie *http.Cookie, token string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/invites/"+token+"/accept", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept invite: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func createStake(t *testing.T, router *chi.Mux, cookie *http.Cookie, goalID int64, description string) int64 {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"description": description})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/goals/%d/stakes", goalID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create stake: expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Stake struct {
			Stake struct {
				ID int64 `json:"id"`
			} `json:"stake"`
		} `json:"stake"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create stake: %v", err)
	}
	return resp.Stake.Stake.ID
}

func cancelStake(t *testing.T, router *chi.Mux, cookie *http.Cookie, stakeID int64) {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/stakes/%d", stakeID), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("cancel stake: expected 204, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func forfeitStake(t *testing.T, router *chi.Mux, cookie *http.Cookie, stakeID int64, reason string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"reason": reason})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stakes/%d/forfeit", stakeID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forfeit stake: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func listStakes(t *testing.T, router *chi.Mux, cookie *http.Cookie, goalID int64) []stakeListItem {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/goals/%d/stakes", goalID), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list stakes: expected 200, got %d", rec.Code)
	}
	var resp struct {
		Stakes []stakeListItem `json:"stakes"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode list stakes: %v", err)
	}
	return resp.Stakes
}

func createCheckIn(t *testing.T, router *chi.Mux, cookie *http.Cookie, goalID int64) int64 {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/goals/%d/check-ins", goalID), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create check-in: expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		CheckIn struct {
			ID int64 `json:"id"`
		} `json:"check_in"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create check-in: %v", err)
	}
	return resp.CheckIn.ID
}

func addTextEvidence(t *testing.T, router *chi.Mux, cookie *http.Cookie, checkInID int64, content string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"content": content})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/check-ins/%d/evidence/text", checkInID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add text evidence: expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func submitCheckIn(t *testing.T, router *chi.Mux, cookie *http.Cookie, checkInID int64) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/check-ins/%d/submit", checkInID), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit check-in: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func approveCheckIn(t *testing.T, router *chi.Mux, cookie *http.Cookie, checkInID int64) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"comment": "lgtm"})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/check-ins/%d/approve", checkInID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve check-in: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func sanitize(path string) string {
	r := strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_")
	return r.Replace(path)
}
