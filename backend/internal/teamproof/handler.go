package teamproof

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for team proofs.
type Handler struct {
	service *Service
}

// NewHandler constructs a Handler over the given Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs all team-proof endpoints.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/teams/{teamID}/proofs", h.handleSubmit)
	r.Get("/teams/{teamID}/feed", h.handleFeed)
	r.Post("/proofs/{proofID}/approve", h.handleApprove)
	r.Post("/proofs/{proofID}/reject", h.handleReject)
	r.Get("/proofs/{proofID}/comments", h.handleListComments)
	r.Post("/proofs/{proofID}/comments", h.handleComment)
}

// DTOs.

type submitInput struct {
	GoalID           int64           `json:"goal_id"`
	DailyLogEntryIDs []int64         `json:"daily_log_entry_ids,omitempty"`
	ProofText        string          `json:"proof_text"`
	Evidence         []EvidenceInput `json:"evidence,omitempty"`
}

type rejectInput struct {
	Comment string `json:"comment"`
}

type proofDTO struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	GoalID      int64      `json:"goal_id"`
	OwnerUserID int64      `json:"owner_user_id"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

type feedItemDTO struct {
	ID            int64     `json:"id"`
	OwnerUserID   int64     `json:"owner_user_id"`
	OwnerAlias    string    `json:"owner_alias"`
	GoalTitle     string    `json:"goal_title"`
	Status        string    `json:"status"`
	CommentsCount int       `json:"comments_count"`
	SubmittedAt   time.Time `json:"submitted_at"`
	CanApprove    bool      `json:"can_approve"`
}

type commentInput struct {
	Text string `json:"text"`
}

type commentDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	UserAlias string    `json:"user_alias"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func toFeedItemDTO(f FeedItem) feedItemDTO {
	return feedItemDTO{
		ID:            f.ID,
		OwnerUserID:   f.OwnerUserID,
		OwnerAlias:    f.OwnerAlias,
		GoalTitle:     f.GoalTitle,
		Status:        f.Status,
		CommentsCount: f.CommentsCount,
		SubmittedAt:   f.SubmittedAt,
		CanApprove:    f.CanApprove,
	}
}

func toCommentDTO(c *Comment) commentDTO {
	return commentDTO{
		ID:        c.ID,
		UserID:    c.UserID,
		UserAlias: c.UserAlias,
		Text:      c.Text,
		CreatedAt: c.CreatedAt,
	}
}

// Handlers.

func (h *Handler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	var in submitInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}

	proof, err := h.service.SubmitProof(r.Context(), SubmitInput{
		TeamID:           teamID,
		OwnerUserID:      user.ID,
		GoalID:           in.GoalID,
		DailyLogEntryIDs: in.DailyLogEntryIDs,
		ProofText:        in.ProofText,
		Evidence:         in.Evidence,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"proof_id": proof.ID,
			"status":   proof.Status,
			"team_id":  proof.TeamID,
		},
	})
}

func (h *Handler) handleFeed(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}

	cursor := int64(0)
	if c := r.URL.Query().Get("cursor"); c != "" {
		if v, err := strconv.ParseInt(c, 10, 64); err == nil {
			cursor = v
		}
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	items, err := h.service.ListFeed(r.Context(), teamID, user.ID, cursor, limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	out := make([]feedItemDTO, 0, len(items))
	for _, f := range items {
		out = append(out, toFeedItemDTO(f))
	}

	nextCursor := int64(0)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"items":       out,
			"next_cursor": nextCursor,
		},
	})
}

func (h *Handler) handleApprove(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	proofID, ok := pathInt64(w, r, "proofID")
	if !ok {
		return
	}
	if err := h.service.ApproveProof(r.Context(), user.ID, proofID); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"approved": true}})
}

func (h *Handler) handleReject(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	proofID, ok := pathInt64(w, r, "proofID")
	if !ok {
		return
	}
	var in rejectInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	if err := h.service.RejectProof(r.Context(), user.ID, proofID, in.Comment); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"rejected": true}})
}

func (h *Handler) handleListComments(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	proofID, ok := pathInt64(w, r, "proofID")
	if !ok {
		return
	}
	comments, err := h.service.ListComments(r.Context(), proofID, user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	out := make([]commentDTO, 0, len(comments))
	for _, c := range comments {
		out = append(out, toCommentDTO(&c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"comments": out}})
}

func (h *Handler) handleComment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	proofID, ok := pathInt64(w, r, "proofID")
	if !ok {
		return
	}
	var in commentInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}
	comment, err := h.service.AddComment(r.Context(), proofID, user.ID, in.Text)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": toCommentDTO(comment)})
}

// Helpers.

func requireAuth(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	user, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrorCode(w, http.StatusUnauthorized, "user.not_authenticated", "Authentication required")
		return users.User{}, false
	}
	return user, true
}

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

var errMap = map[error]struct {
	status int
	code   string
}{
	ErrProofNotFound:         {http.StatusNotFound, "proof.not_found"},
	ErrNotAuthor:             {http.StatusForbidden, "proof.not_author"},
	ErrNotApprover:           {http.StatusForbidden, "proof.not_approver"},
	ErrCannotReviewSelf:      {http.StatusForbidden, "proof.cannot_review_self"},
	ErrAlreadyReviewed:       {http.StatusConflict, "proof.already_reviewed"},
	ErrNoArtifact:            {http.StatusBadRequest, "proof.no_artifact"},
	ErrInvalidGoal:           {http.StatusBadRequest, "proof.invalid_goal"},
	ErrEntriesNotOwned:       {http.StatusBadRequest, "proof.entries_not_owned"},
	ErrEntryAlreadyUsed:      {http.StatusConflict, "proof.entry_already_used"},
	ErrRejectCommentRequired: {http.StatusBadRequest, "proof.reject_comment_required"},
	ErrCannotComment:         {http.StatusForbidden, "proof.cannot_comment"},
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
