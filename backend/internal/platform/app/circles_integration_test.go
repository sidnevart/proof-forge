package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

func TestCircleWeeklyAssemblyFlow(t *testing.T) {
	pool := testutil.OpenIntegrationPool(t)

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

	ownerCookie := registerSession(t, router, "owner@example.com", "Owner")
	peerCookie := registerSession(t, router, "peer@example.com", "Peer")

	circleResp := apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/circles",
		ownerCookie,
		map[string]any{
			"name": "Kotlin growth ring",
		},
		http.StatusCreated,
	)

	circleID := int64(circleResp["circle"].(map[string]any)["id"].(float64))
	inviteCode := circleResp["circle"].(map[string]any)["invite_code"].(string)

	apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/circles/join",
		peerCookie,
		map[string]any{
			"invite_code": inviteCode,
		},
		http.StatusOK,
	)

	goalResp := apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/goals",
		ownerCookie,
		map[string]any{
			"title":       "Изучить Kotlin через маленькие сдвиги",
			"description": "Каждую неделю показывать реальный шаг",
			"buddy_name":  "Peer",
			"buddy_email": "peer@example.com",
			"circle_id":   circleID,
		},
		http.StatusCreated,
	)

	goalID := int64(goalResp["goal"].(map[string]any)["goal"].(map[string]any)["id"].(float64))
	inviteToken := goalResp["goal"].(map[string]any)["invite"].(map[string]any)["acceptance_token"].(string)

	apiJSON(t, router, http.MethodPost, "/v1/invites/"+inviteToken+"/accept", peerCookie, nil, http.StatusOK)

	checkInResp := apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/goals/"+itoa(goalID)+"/check-ins",
		ownerCookie,
		nil,
		http.StatusCreated,
	)
	checkInID := int64(checkInResp["check_in"].(map[string]any)["id"].(float64))

	apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/check-ins/"+itoa(checkInID)+"/evidence/text",
		ownerCookie,
		map[string]any{
			"content": "Разобрал data classes и написал маленький CLI.",
		},
		http.StatusCreated,
	)

	apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/check-ins/"+itoa(checkInID)+"/submit",
		ownerCookie,
		nil,
		http.StatusOK,
	)

	apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/check-ins/"+itoa(checkInID)+"/approve",
		peerCookie,
		map[string]any{
			"comment": "Подтверждаю шаг",
		},
		http.StatusOK,
	)

	standingsResp := apiJSON(
		t,
		router,
		http.MethodGet,
		"/v1/circles/"+itoa(circleID)+"/standings",
		ownerCookie,
		nil,
		http.StatusOK,
	)

	entries := standingsResp["standings"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 standing entries, got %d", len(entries))
	}

	first := entries[0].(map[string]any)
	second := entries[1].(map[string]any)
	if first["user_email"] != "owner@example.com" {
		t.Fatalf("expected owner to lead standings, got %v", first["user_email"])
	}
	if first["weekly_status"] != "approved" {
		t.Fatalf("expected owner weekly status approved, got %v", first["weekly_status"])
	}
	if second["weekly_status"] != "at_risk" {
		t.Fatalf("expected peer weekly status at_risk, got %v", second["weekly_status"])
	}

	assemblyResp := apiJSON(
		t,
		router,
		http.MethodGet,
		"/v1/circles/"+itoa(circleID)+"/weekly-assembly",
		ownerCookie,
		nil,
		http.StatusOK,
	)

	if assemblyResp["circle"].(map[string]any)["id"].(float64) != float64(circleID) {
		t.Fatalf("expected weekly assembly for circle %d", circleID)
	}
	events := assemblyResp["events"].([]any)
	if len(events) == 0 {
		t.Fatal("expected weekly assembly events")
	}
}

func registerSession(t *testing.T, router http.Handler, email string, displayName string) *http.Cookie {
	t.Helper()

	resp := apiJSON(
		t,
		router,
		http.MethodPost,
		"/v1/register",
		nil,
		map[string]any{
			"email":        email,
			"display_name": displayName,
		},
		http.StatusCreated,
	)
	_ = resp

	rec := lastRecorder
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "pf_session" {
			return cookie
		}
	}
	t.Fatalf("expected session cookie for %s", email)
	return nil
}

var lastRecorder *httptest.ResponseRecorder

func apiJSON(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	cookie *http.Cookie,
	body any,
	wantStatus int,
) map[string]any {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	lastRecorder = rec
	router.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("%s %s: expected status %d, got %d: %s", method, path, wantStatus, rec.Code, rec.Body.String())
	}

	if rec.Body.Len() == 0 {
		return map[string]any{}
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response for %s %s: %v", method, path, err)
	}
	return payload
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
