package teams

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for team management. All routes require
// an authenticated user — the auth middleware is mounted by the app builder.
type Handler struct {
	service *Service
}

// NewHandler constructs a Handler over the given Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs all /teams/* endpoints on the given router. The
// router is expected to be already wrapped with the auth middleware that
// populates users.CurrentUser(ctx).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/teams", h.handleCreate)
	r.Get("/teams", h.handleList)
	r.Get("/teams/{teamID}", h.handleGet)
	r.Post("/teams/{teamID}/archive", h.handleArchive)
	r.Post("/teams/{teamID}/regenerate-invite", h.handleRegenerateInvite)
	r.Post("/teams/join", h.handleJoin)
	r.Post("/teams/{teamID}/leave", h.handleLeave)
	r.Post("/teams/{teamID}/members/{userID}/role", h.handleChangeRole)
	r.Delete("/teams/{teamID}/members/{userID}", h.handleRemoveMember)
	r.Post("/teams/{teamID}/members/{userID}/ai-consent", h.handleSetAIConsent)
}

// ──────────────────────────────────────────────────────────────────────
// DTOs (request bodies + response shapes)
// ──────────────────────────────────────────────────────────────────────

type createInput struct {
	Name   string `json:"name"`
	AIMode string `json:"ai_mode,omitempty"`
}

type joinInput struct {
	InviteCode string `json:"invite_code"`
}

type changeRoleInput struct {
	Role string `json:"role"`
}

type aiConsentInput struct {
	Consent bool `json:"consent"`
}

type teamDTO struct {
	ID          int64      `json:"id"`
	LeadUserID  int64      `json:"lead_user_id"`
	Name        string     `json:"name"`
	InviteCode  string     `json:"invite_code,omitempty"`
	MemberLimit int        `json:"member_limit"`
	AIMode      string     `json:"ai_mode"`
	CreatedAt   time.Time  `json:"created_at"`
	ArchivedAt  *time.Time `json:"archived_at"`
}

type membershipDTO struct {
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	AIConsent bool      `json:"ai_consent"`
	Timezone  string    `json:"timezone"`
	JoinedAt  time.Time `json:"joined_at"`
}

type detailDTO struct {
	Team         teamDTO       `json:"team"`
	MyMembership membershipDTO `json:"my_membership"`
	MemberCount  int           `json:"member_count"`
}

func toDetailDTO(d Detail) detailDTO {
	inviteCode := ""
	if d.MyMembership.Role == RoleLead {
		inviteCode = d.Team.InviteCode
	}
	return detailDTO{
		Team: teamDTO{
			ID:          d.Team.ID,
			LeadUserID:  d.Team.LeadUserID,
			Name:        d.Team.Name,
			InviteCode:  inviteCode,
			MemberLimit: d.Team.MemberLimit,
			AIMode:      string(d.Team.AIMode),
			CreatedAt:   d.Team.CreatedAt,
			ArchivedAt:  d.Team.ArchivedAt,
		},
		MyMembership: membershipDTO{
			TeamID:    d.MyMembership.TeamID,
			UserID:    d.MyMembership.UserID,
			Role:      string(d.MyMembership.Role),
			Status:    string(d.MyMembership.Status),
			AIConsent: d.MyMembership.AIConsent,
			Timezone:  d.MyMembership.Timezone,
			JoinedAt:  d.MyMembership.JoinedAt,
		},
		MemberCount: d.MemberCount,
	}
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	var in createInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	d, err := h.service.CreateTeam(r.Context(), user.ID, in.Name, AIMode(in.AIMode))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toDetailDTO(d)})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teams, err := h.service.ListMyTeams(r.Context(), user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]detailDTO, 0, len(teams))
	for _, d := range teams {
		out = append(out, toDetailDTO(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"teams": out}})
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	d, err := h.service.GetTeamForUser(r.Context(), teamID, user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toDetailDTO(d)})
}

func (h *Handler) handleArchive(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	if err := h.service.ArchiveTeam(r.Context(), user.ID, teamID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"archived": true}})
}

func (h *Handler) handleRegenerateInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	code, err := h.service.RegenerateInvite(r.Context(), user.ID, teamID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"invite_code": code}})
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	var in joinInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	d, err := h.service.JoinTeam(r.Context(), user.ID, in.InviteCode)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toDetailDTO(d)})
}

func (h *Handler) handleLeave(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	if err := h.service.LeaveTeam(r.Context(), user.ID, teamID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"left": true}})
}

func (h *Handler) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	targetUserID, ok := pathInt64(w, r, "userID")
	if !ok {
		return
	}
	var in changeRoleInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	role, err := ParseRole(in.Role)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if err := h.service.ChangeMemberRole(r.Context(), user.ID, teamID, targetUserID, role); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"role": string(role)}})
}

func (h *Handler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	targetUserID, ok := pathInt64(w, r, "userID")
	if !ok {
		return
	}
	if err := h.service.RemoveMember(r.Context(), user.ID, teamID, targetUserID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"removed": true}})
}

func (h *Handler) handleSetAIConsent(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	targetUserID, ok := pathInt64(w, r, "userID")
	if !ok {
		return
	}
	var in aiConsentInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	if err := h.service.SetMyAIConsent(r.Context(), user.ID, teamID, targetUserID, in.Consent); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ai_consent": in.Consent}})
}

// ──────────────────────────────────────────────────────────────────────
// Helpers — local to teams package, do not export.
// ──────────────────────────────────────────────────────────────────────

func requireAuth(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	user, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrorCode(w, http.StatusUnauthorized, "user.not_authenticated", "Authentication required")
		return users.User{}, false
	}
	return user, true
}

// decodeJSON returns true on success, false if it has already written an
// error response. Naming is a bit awkward (`if err := decode...; !err`) but
// reads as "if not OK, bail" which is the intent.
func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "Request body must be valid JSON")
		return false
	}
	return true
}

func pathInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := chi.URLParam(r, key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "Path parameter must be a positive integer")
		return 0, false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorCode(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// errMap is the closed list of domain-error → (status, code) mappings used
// by writeDomainError. The list lives here (and not in domain.go) because
// HTTP status codes are an HTTP-layer concern.
var errMap = map[error]struct {
	status int
	code   string
}{
	ErrTeamNotFound:                {http.StatusNotFound, "team.not_found"},
	ErrTeamArchived:                {http.StatusConflict, "team.archived"},
	ErrTeamFull:                    {http.StatusConflict, "team.full"},
	ErrAlreadyMember:               {http.StatusConflict, "team.already_member"},
	ErrNotMember:                   {http.StatusForbidden, "team.not_member"},
	ErrNotLead:                     {http.StatusForbidden, "team.not_lead"},
	ErrMustHaveLead:                {http.StatusConflict, "team.must_have_lead"},
	ErrCannotDemoteSelf:            {http.StatusConflict, "team.cannot_demote_self"},
	ErrCannotRemoveSelfAsLead:      {http.StatusConflict, "team.cannot_remove_self_as_lead"},
	ErrCannotLeaveAsOnlyLead:       {http.StatusConflict, "team.cannot_leave_as_only_lead"},
	ErrCannotChangeOthersAIConsent: {http.StatusForbidden, "team.cannot_change_others_consent"},
	ErrInvalidInviteCode:           {http.StatusNotFound, "invite.invalid"},
	ErrInvalidTeamName:             {http.StatusBadRequest, "validation.field_invalid"},
	ErrInvalidRole:                 {http.StatusBadRequest, "validation.field_invalid"},
	ErrInvalidAIMode:               {http.StatusBadRequest, "validation.field_invalid"},
	ErrInvalidTimezone:             {http.StatusBadRequest, "validation.field_invalid"},
}

func writeDomainError(w http.ResponseWriter, err error) {
	for sentinel, mapping := range errMap {
		if errors.Is(err, sentinel) {
			writeErrorCode(w, mapping.status, mapping.code, sentinel.Error())
			return
		}
	}
	writeErrorCode(w, http.StatusInternalServerError, "internal.unexpected", "An unexpected error occurred")
}

// ensure context types stay aligned with the rest of the codebase.
var _ = func() context.Context { return context.Background() }
