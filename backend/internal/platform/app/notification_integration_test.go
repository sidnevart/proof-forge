package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sidnevart/proof-forge/backend/testutil"
)

// TestCheckinSubmitTriggersBuddyApprovalTelegram verifies that submitting a
// check-in writes a "checkin.submitted" domain event and that NudgeEngine.Tick
// delivers an approval-keyboard Telegram message to the buddy's linked chat.
func TestCheckinSubmitTriggersBuddyApprovalTelegram(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)

	tgMock := &mockTelegramSender{}
	emailMock := &mockEmailSender{}
	router, engine := buildNotifRouter(t, pool, tgMock, emailMock)

	// Register owner and buddy.
	ownerCookie := registerSession(t, router, "owner-tg@example.com", "Владелец")
	buddyCookie := registerSession(t, router, "buddy-tg@example.com", "Партнёр")

	// Link buddy's Telegram account (chat ID 99001).
	buddyID := getUserIDByEmail(t, pool, "buddy-tg@example.com")
	const buddyChatID = int64(99001)
	linkTelegram(t, pool, buddyID, buddyChatID)

	// Create goal (owner sends invite to buddy).
	goalResp := apiJSON(t, router, http.MethodPost, "/v1/goals", ownerCookie, map[string]any{
		"title":         "Тестовая цель для пруфа",
		"description":   "Описание",
		"buddy_name":    "Партнёр",
		"buddy_email":   "buddy-tg@example.com",
		"proof_examples": "скриншот, запись",
	}, http.StatusCreated)

	inviteToken := goalResp["goal"].(map[string]any)["invite"].(map[string]any)["acceptance_token"].(string)
	goalID := int64(goalResp["goal"].(map[string]any)["goal"].(map[string]any)["id"].(float64))

	// Buddy accepts the invite → pact becomes active.
	apiJSON(t, router, http.MethodPost, "/v1/invites/"+inviteToken+"/accept", buddyCookie, nil, http.StatusOK)

	// Owner creates a check-in and submits it with evidence.
	ciResp := apiJSON(t, router, http.MethodPost, "/v1/goals/"+itoa(goalID)+"/check-ins", ownerCookie, nil, http.StatusCreated)
	checkInID := int64(ciResp["check_in"].(map[string]any)["id"].(float64))

	apiJSON(t, router, http.MethodPost, "/v1/check-ins/"+itoa(checkInID)+"/evidence/text", ownerCookie, map[string]any{
		"content": "Сделал первый шаг, вот скриншот.",
	}, http.StatusCreated)

	apiJSON(t, router, http.MethodPost, "/v1/check-ins/"+itoa(checkInID)+"/submit", ownerCookie, nil, http.StatusOK)

	// domain_event "checkin.submitted" must be written and unprocessed.
	if n := countUnprocessedEvents(t, pool, "checkin.submitted"); n != 1 {
		t.Fatalf("expected 1 unprocessed checkin.submitted event, got %d", n)
	}

	// Run NudgeEngine tick — it should process the event and send a TG keyboard.
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatalf("engine.Tick: %v", err)
	}

	// Verify keyboard message sent to buddy's chat.
	kb := tgMock.FindKeyboardMessage()
	if kb == nil {
		t.Fatal("expected an approval keyboard message to be sent to buddy, got none")
	}
	if kb.chatID != buddyChatID {
		t.Fatalf("expected keyboard message to chatID %d, got %d", buddyChatID, kb.chatID)
	}
	if !strings.Contains(kb.text, "НОВЫЙ ПРУФ") {
		t.Fatalf("expected keyboard message text to contain 'НОВЫЙ ПРУФ', got: %q", kb.text)
	}

	// Verify callback data encodes the correct check-in ID.
	if len(kb.keyboard.InlineKeyboard) == 0 || len(kb.keyboard.InlineKeyboard[0]) == 0 {
		t.Fatal("expected non-empty inline keyboard")
	}
	wantApproveData := fmt.Sprintf("review_%d_approve", checkInID)
	gotApproveData := kb.keyboard.InlineKeyboard[0][0].CallbackData
	if gotApproveData != wantApproveData {
		t.Fatalf("expected approve button callback_data %q, got %q", wantApproveData, gotApproveData)
	}
}

// TestBuddyApprovesViaWebhookOwnerNotified verifies the full approval loop:
// buddy taps approve in Telegram → checkin.approved event → NudgeEngine sends
// "ОДОБРЕНО" message to owner's Telegram.
func TestBuddyApprovesViaWebhookOwnerNotified(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)

	tgMock := &mockTelegramSender{}
	emailMock := &mockEmailSender{}
	router, engine := buildNotifRouter(t, pool, tgMock, emailMock)

	ownerCookie := registerSession(t, router, "owner-approve@example.com", "Владелец")
	buddyCookie := registerSession(t, router, "buddy-approve@example.com", "Партнёр")

	ownerID := getUserIDByEmail(t, pool, "owner-approve@example.com")
	buddyID := getUserIDByEmail(t, pool, "buddy-approve@example.com")
	const ownerChatID = int64(99002)
	const buddyChatID = int64(99003)
	linkTelegram(t, pool, ownerID, ownerChatID)
	linkTelegram(t, pool, buddyID, buddyChatID)

	// Create goal + accept invite + create + submit check-in.
	goalResp := apiJSON(t, router, http.MethodPost, "/v1/goals", ownerCookie, map[string]any{
		"title":          "Цель для теста одобрения",
		"description":    "Описание",
		"buddy_name":     "Партнёр",
		"buddy_email":    "buddy-approve@example.com",
		"proof_examples": "скриншот",
	}, http.StatusCreated)

	inviteToken := goalResp["goal"].(map[string]any)["invite"].(map[string]any)["acceptance_token"].(string)
	goalID := int64(goalResp["goal"].(map[string]any)["goal"].(map[string]any)["id"].(float64))

	apiJSON(t, router, http.MethodPost, "/v1/invites/"+inviteToken+"/accept", buddyCookie, nil, http.StatusOK)

	ciResp := apiJSON(t, router, http.MethodPost, "/v1/goals/"+itoa(goalID)+"/check-ins", ownerCookie, nil, http.StatusCreated)
	checkInID := int64(ciResp["check_in"].(map[string]any)["id"].(float64))

	apiJSON(t, router, http.MethodPost, "/v1/check-ins/"+itoa(checkInID)+"/evidence/text", ownerCookie, map[string]any{
		"content": "Готово.",
	}, http.StatusCreated)

	apiJSON(t, router, http.MethodPost, "/v1/check-ins/"+itoa(checkInID)+"/submit", ownerCookie, nil, http.StatusOK)

	// First tick: process checkin.submitted → send approval keyboard to buddy.
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatalf("engine.Tick (submitted): %v", err)
	}

	// Simulate Telegram callback: buddy taps ✅ ОДОБРИТЬ.
	update := map[string]any{
		"update_id": 1001,
		"callback_query": map[string]any{
			"id":   "cq_test_1",
			"from": map[string]any{"id": buddyChatID, "first_name": "Партнёр"},
			"message": map[string]any{
				"message_id": 42,
				"chat":       map[string]any{"id": buddyChatID},
			},
			"data": fmt.Sprintf("review_%d_approve", checkInID),
		},
	}
	body, _ := json.Marshal(update)

	apiRaw(t, router, http.MethodPost, "/telegram/webhook", nil, body, http.StatusOK)

	// domain_event "checkin.approved" must be written.
	if n := countUnprocessedEvents(t, pool, "checkin.approved"); n < 1 {
		t.Fatal("expected checkin.approved domain event after Telegram callback")
	}

	// Second tick: process checkin.approved → notify owner.
	tgMock.Reset()
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatalf("engine.Tick (approved): %v", err)
	}

	// Owner should receive "ОДОБРЕНО" message.
	ownerMsgs := tgMock.FindMessagesForChat(ownerChatID)
	if len(ownerMsgs) == 0 {
		t.Fatal("expected at least one Telegram message sent to owner after approval")
	}
	found := false
	for _, msg := range ownerMsgs {
		if strings.Contains(msg.text, "ОДОБРЕНО") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected message containing 'ОДОБРЕНО' for owner chat %d, got: %v", ownerChatID, ownerMsgs)
	}
}

// TestAcceptInviteSendsOwnerEmailAndTelegram verifies that when a buddy accepts
// an invite, the goal owner receives:
//  1. A "buddy accepted" email (BuddyAcceptedParams)
//  2. A Telegram "ПРИНЯЛ ПРИГЛАШЕНИЕ" nudge via NudgeEngine
func TestAcceptInviteSendsOwnerEmailAndTelegram(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)

	tgMock := &mockTelegramSender{}
	emailMock := &mockEmailSender{}
	router, engine := buildNotifRouter(t, pool, tgMock, emailMock)

	ownerCookie := registerSession(t, router, "owner-accept@example.com", "Владелец")
	buddyCookie := registerSession(t, router, "buddy-accept@example.com", "Партнёр")

	ownerID := getUserIDByEmail(t, pool, "owner-accept@example.com")
	const ownerChatID = int64(99004)
	linkTelegram(t, pool, ownerID, ownerChatID)

	// Create goal.
	goalResp := apiJSON(t, router, http.MethodPost, "/v1/goals", ownerCookie, map[string]any{
		"title":          "Цель для теста принятия",
		"description":    "Описание",
		"buddy_name":     "Партнёр",
		"buddy_email":    "buddy-accept@example.com",
		"proof_examples": "скриншот",
	}, http.StatusCreated)

	inviteToken := goalResp["goal"].(map[string]any)["invite"].(map[string]any)["acceptance_token"].(string)

	// Reset to track only the email from AcceptInvite (not CreateGoal).
	emailMock.Reset()

	// Buddy accepts invite.
	apiJSON(t, router, http.MethodPost, "/v1/invites/"+inviteToken+"/accept", buddyCookie, nil, http.StatusOK)

	// 1. Verify "buddy accepted" email was sent to owner.
	accepted := emailMock.LastBuddyAccepted()
	if accepted == nil {
		t.Fatal("expected SendBuddyAccepted to be called after invite acceptance, but got none")
	}
	if accepted.To != "owner-accept@example.com" {
		t.Fatalf("expected email To owner-accept@example.com, got %q", accepted.To)
	}
	if accepted.BuddyName != "Партнёр" {
		t.Fatalf("expected BuddyName 'Партнёр', got %q", accepted.BuddyName)
	}
	if accepted.GoalTitle != "Цель для теста принятия" {
		t.Fatalf("expected GoalTitle 'Цель для теста принятия', got %q", accepted.GoalTitle)
	}

	// 2. Verify "invite.accepted" domain event written.
	if n := countUnprocessedEvents(t, pool, "invite.accepted"); n < 1 {
		t.Fatal("expected invite.accepted domain event to be written")
	}

	// 3. NudgeEngine tick → sends TG message to owner.
	if err := engine.Tick(context.Background()); err != nil {
		t.Fatalf("engine.Tick: %v", err)
	}

	ownerMsgs := tgMock.FindMessagesForChat(ownerChatID)
	if len(ownerMsgs) == 0 {
		t.Fatal("expected Telegram message to owner after invite accepted")
	}
	found := false
	for _, msg := range ownerMsgs {
		if strings.Contains(msg.text, "ПРИНЯЛ ПРИГЛАШЕНИЕ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected message containing 'ПРИНЯЛ ПРИГЛАШЕНИЕ' for owner, got: %v", ownerMsgs)
	}
}

// ─── raw HTTP helper for non-JSON endpoints (e.g. /telegram/webhook) ─────────

func apiRaw(t *testing.T, router http.Handler, method, path string, cookie *http.Cookie, body []byte, wantStatus int) []byte {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s: expected status %d, got %d: %s", method, path, wantStatus, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}
