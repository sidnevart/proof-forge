package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platformconfig "github.com/sidnevart/proof-forge/backend/internal/platform/config"
	"github.com/sidnevart/proof-forge/backend/internal/platform/httpx"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
	"github.com/sidnevart/proof-forge/backend/testutil"
)

func TestRegistrationGoalCreationAndDashboardFlow(t *testing.T) {
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

	registerBody, _ := json.Marshal(map[string]any{
		"email":        "owner@example.com",
		"display_name": "Owner",
	})
	registerReq := httptest.NewRequest(http.MethodPost, "/v1/register", bytes.NewReader(registerBody))
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerRec.Code)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range registerRec.Result().Cookies() {
		if cookie.Name == "pf_session" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie after registration")
	}

	createGoalBody, _ := json.Marshal(map[string]any{
		"title":       "Ship MVP vertical slice",
		"description": "Registration + goals dashboard",
		"buddy_name":  "Serious Peer",
		"buddy_email": "peer@example.com",
	})
	createGoalReq := httptest.NewRequest(http.MethodPost, "/v1/goals", bytes.NewReader(createGoalBody))
	createGoalReq.AddCookie(sessionCookie)
	createGoalRec := httptest.NewRecorder()
	router.ServeHTTP(createGoalRec, createGoalReq)

	if createGoalRec.Code != http.StatusCreated {
		t.Fatalf("expected goal creation status 201, got %d", createGoalRec.Code)
	}

	var createGoalResp struct {
		Goal struct {
			Goal struct {
				Status string `json:"status"`
				Title  string `json:"title"`
			} `json:"goal"`
		} `json:"goal"`
	}
	if err := json.NewDecoder(createGoalRec.Body).Decode(&createGoalResp); err != nil {
		t.Fatalf("decode create goal response: %v", err)
	}
	if createGoalResp.Goal.Goal.Status != "pending_buddy_acceptance" {
		t.Fatalf("expected pending_buddy_acceptance, got %q", createGoalResp.Goal.Goal.Status)
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	dashboardReq.AddCookie(sessionCookie)
	dashboardRec := httptest.NewRecorder()
	router.ServeHTTP(dashboardRec, dashboardReq)

	if dashboardRec.Code != http.StatusOK {
		t.Fatalf("expected dashboard status 200, got %d", dashboardRec.Code)
	}

	var dashboardResp struct {
		Summary struct {
			TotalGoals             int `json:"total_goals"`
			PendingBuddyAcceptance int `json:"pending_buddy_acceptance"`
		} `json:"summary"`
		Goals []any `json:"goals"`
	}
	if err := json.NewDecoder(dashboardRec.Body).Decode(&dashboardResp); err != nil {
		t.Fatalf("decode dashboard response: %v", err)
	}
	if dashboardResp.Summary.TotalGoals != 1 {
		t.Fatalf("expected total goals 1, got %d", dashboardResp.Summary.TotalGoals)
	}
	if dashboardResp.Summary.PendingBuddyAcceptance != 1 {
		t.Fatalf("expected pending buddy acceptance 1, got %d", dashboardResp.Summary.PendingBuddyAcceptance)
	}
	if len(dashboardResp.Goals) != 1 {
		t.Fatalf("expected 1 goal in dashboard, got %d", len(dashboardResp.Goals))
	}
}
