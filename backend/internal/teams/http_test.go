package teams

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// authMW is a tiny middleware that injects a fake authenticated user into
// the request context — same shape that the production middleware writes.
func authMW(userID int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := users.WithAuthenticatedUser(r.Context(), users.User{ID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newHandlerStack(t *testing.T, userID int64) (*Handler, http.Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	svc := NewService(repo)
	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(authMW(userID))
	handler.RegisterRoutes(r)
	return handler, r, repo
}

func do(r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rr.Body.String())
	}
	return m
}

func errorCode(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	body := decodeBody(t, rr)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error object in body: %s", rr.Body.String())
	}
	code, _ := errObj["code"].(string)
	return code
}

// ──────────────────────────────────────────────────────────────────────
// Handler tests
// ──────────────────────────────────────────────────────────────────────

func TestHandler_CreateTeam_201(t *testing.T) {
	_, r, _ := newHandlerStack(t, 7)

	rr := do(r, "POST", "/teams", map[string]any{"name": "ML team", "ai_mode": "metadata-only"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	data := body["data"].(map[string]any)
	team := data["team"].(map[string]any)
	if team["name"] != "ML team" {
		t.Errorf("team.name = %v, want ML team", team["name"])
	}
	if team["invite_code"] == "" {
		t.Errorf("lead response must include invite_code")
	}
	mem := data["my_membership"].(map[string]any)
	if mem["role"] != "lead" {
		t.Errorf("my_membership.role = %v, want lead", mem["role"])
	}
	if data["member_count"].(float64) != 1 {
		t.Errorf("member_count = %v, want 1", data["member_count"])
	}
}

func TestHandler_CreateTeam_ValidationFailure_400(t *testing.T) {
	_, r, _ := newHandlerStack(t, 7)
	rr := do(r, "POST", "/teams", map[string]any{"name": "  "})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if errorCode(t, rr) != "validation.field_invalid" {
		t.Errorf("error.code = %q, want validation.field_invalid", errorCode(t, rr))
	}
}

func TestHandler_CreateTeam_BadJSON_400(t *testing.T) {
	_, r, _ := newHandlerStack(t, 7)
	req := httptest.NewRequest("POST", "/teams", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_JoinTeam_201(t *testing.T) {
	// Lead creates team via service, member joins via HTTP.
	leadHandler, _, repo := newHandlerStack(t, 1)
	_ = leadHandler
	svc := NewService(repo)
	d, err := svc.CreateTeam(httptest.NewRequest("POST", "/", nil).Context(), 1, "T", "")
	if err != nil {
		t.Fatal(err)
	}

	// Now do the join as user 8 (separate handler stack with a fresh user id,
	// but the same repo behind the service).
	memberSvc := NewService(repo)
	memberHandler := NewHandler(memberSvc)
	r := chi.NewRouter()
	r.Use(authMW(8))
	memberHandler.RegisterRoutes(r)

	rr := do(r, "POST", "/teams/join", map[string]any{"invite_code": d.Team.InviteCode})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	mem := body["data"].(map[string]any)["my_membership"].(map[string]any)
	if mem["role"] != "member" {
		t.Errorf("role = %v, want member", mem["role"])
	}
	team := body["data"].(map[string]any)["team"].(map[string]any)
	if _, ok := team["invite_code"]; ok {
		t.Errorf("member join response must not expose invite_code")
	}
}

func TestHandler_JoinTeam_BadCode_404(t *testing.T) {
	_, r, _ := newHandlerStack(t, 7)
	rr := do(r, "POST", "/teams/join", map[string]any{"invite_code": "NOPE"})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if errorCode(t, rr) != "invite.invalid" {
		t.Errorf("error.code = %q, want invite.invalid", errorCode(t, rr))
	}
}

func TestHandler_JoinTeam_Archived_409(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH

	svc := NewService(repo)
	d, err := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, d.Team.ID); err != nil {
		t.Fatal(err)
	}

	memberH := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(8))
	memberH.RegisterRoutes(r)

	rr := do(r, "POST", "/teams/join", map[string]any{"invite_code": d.Team.InviteCode})
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	if errorCode(t, rr) != "team.archived" {
		t.Errorf("error.code = %q, want team.archived", errorCode(t, rr))
	}
}

func TestHandler_GetTeam_200(t *testing.T) {
	_, r, _ := newHandlerStack(t, 1)
	// create team via HTTP first to populate the repo through this stack.
	createRR := do(r, "POST", "/teams", map[string]any{"name": "X"})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create failed: %s", createRR.Body.String())
	}
	id := int64(decodeBody(t, createRR)["data"].(map[string]any)["team"].(map[string]any)["id"].(float64))

	rr := do(r, "GET", "/teams/"+strconv.FormatInt(id, 10), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandler_GetTeam_NonMember_403(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH
	svc := NewService(repo)
	d, _ := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")

	otherH := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(99))
	otherH.RegisterRoutes(r)

	rr := do(r, "GET", "/teams/"+strconv.FormatInt(d.Team.ID, 10), nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if errorCode(t, rr) != "team.not_member" {
		t.Errorf("error.code = %q, want team.not_member", errorCode(t, rr))
	}
}

func TestHandler_ChangeRole_NotLead_403(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH
	svc := NewService(repo)
	d, _ := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")
	_, _ = svc.JoinTeam(httptest.NewRequest("GET", "/", nil).Context(), 8, d.Team.InviteCode)

	memberH := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(8))
	memberH.RegisterRoutes(r)

	rr := do(r, "POST", "/teams/"+strconv.FormatInt(d.Team.ID, 10)+"/members/1/role",
		map[string]any{"role": "member"})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if errorCode(t, rr) != "team.not_lead" {
		t.Errorf("error.code = %q, want team.not_lead", errorCode(t, rr))
	}
}

func TestHandler_ArchiveTeam_NotLead_403(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH
	svc := NewService(repo)
	d, _ := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")
	_, _ = svc.JoinTeam(httptest.NewRequest("GET", "/", nil).Context(), 8, d.Team.InviteCode)

	memberH := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(8))
	memberH.RegisterRoutes(r)

	rr := do(r, "POST", "/teams/"+strconv.FormatInt(d.Team.ID, 10)+"/archive", nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if errorCode(t, rr) != "team.not_lead" {
		t.Errorf("error.code = %q, want team.not_lead", errorCode(t, rr))
	}
}

func TestHandler_AICotsentChangeOthers_403(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH
	svc := NewService(repo)
	d, _ := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")
	_, _ = svc.JoinTeam(httptest.NewRequest("GET", "/", nil).Context(), 8, d.Team.InviteCode)

	leadHandler := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(1))
	leadHandler.RegisterRoutes(r)

	// Lead trying to flip member 8's consent — must fail.
	rr := do(r, "POST", "/teams/"+strconv.FormatInt(d.Team.ID, 10)+"/members/8/ai-consent",
		map[string]any{"consent": true})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if errorCode(t, rr) != "team.cannot_change_others_consent" {
		t.Errorf("error.code = %q, want team.cannot_change_others_consent", errorCode(t, rr))
	}
}

func TestHandler_LeaveTeam_OnlyLead_409(t *testing.T) {
	leadH, _, repo := newHandlerStack(t, 1)
	_ = leadH
	svc := NewService(repo)
	d, _ := svc.CreateTeam(httptest.NewRequest("GET", "/", nil).Context(), 1, "T", "")

	leadHandler := NewHandler(NewService(repo))
	r := chi.NewRouter()
	r.Use(authMW(1))
	leadHandler.RegisterRoutes(r)

	rr := do(r, "POST", "/teams/"+strconv.FormatInt(d.Team.ID, 10)+"/leave", nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	if errorCode(t, rr) != "team.cannot_leave_as_only_lead" {
		t.Errorf("error.code = %q, want team.cannot_leave_as_only_lead", errorCode(t, rr))
	}
}

func TestHandler_RegenerateInvite_NewCodeReturned(t *testing.T) {
	_, r, _ := newHandlerStack(t, 1)
	createRR := do(r, "POST", "/teams", map[string]any{"name": "X"})
	team := decodeBody(t, createRR)["data"].(map[string]any)["team"].(map[string]any)
	id := int64(team["id"].(float64))
	oldCode := team["invite_code"].(string)

	rr := do(r, "POST", "/teams/"+strconv.FormatInt(id, 10)+"/regenerate-invite", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := decodeBody(t, rr)
	newCode := body["data"].(map[string]any)["invite_code"].(string)
	if newCode == oldCode {
		t.Errorf("new code %q must differ from old", newCode)
	}
}

func TestHandler_Unauthenticated_401(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	handler := NewHandler(svc)

	r := chi.NewRouter()
	// no authMW
	handler.RegisterRoutes(r)

	rr := do(r, "POST", "/teams", map[string]any{"name": "x"})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	if errorCode(t, rr) != "user.not_authenticated" {
		t.Errorf("error.code = %q, want user.not_authenticated", errorCode(t, rr))
	}
}
