package checkins

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

const maxUploadBytes = MaxFileSizeBytes + 512 // small overage for multipart overhead

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/goals/{goalID}/check-ins", h.handleCreate)
	r.Get("/goals/{goalID}/check-ins", h.handleList)
	r.Get("/check-ins/{checkInID}", h.handleGet)
	r.Post("/check-ins/{checkInID}/submit", h.handleSubmit)
	r.Post("/check-ins/{checkInID}/evidence/text", h.handleAddText)
	r.Post("/check-ins/{checkInID}/evidence/link", h.handleAddLink)
	r.Post("/check-ins/{checkInID}/evidence/file", h.handleAddFile)
	r.Post("/check-ins/{checkInID}/approve", h.handleApprove)
	r.Post("/check-ins/{checkInID}/reject", h.handleReject)
	r.Post("/check-ins/{checkInID}/request-changes", h.handleRequestChanges)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	goalID, ok := pathInt64(w, r, "goalID")
	if !ok {
		return
	}

	ci, err := h.service.CreateCheckIn(r.Context(), actor, goalID)
	if err != nil {
		h.writeServiceError(w, r, "create check-in", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"check_in": ci})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	goalID, ok := pathInt64(w, r, "goalID")
	if !ok {
		return
	}

	list, err := h.service.ListForGoal(r.Context(), actor, goalID)
	if err != nil {
		h.writeServiceError(w, r, "list check-ins", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"check_ins": list})
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	view, err := h.service.GetDetail(r.Context(), actor, checkInID)
	if err != nil {
		h.writeServiceError(w, r, "get check-in", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"check_in": view.CheckIn, "evidence": view.Evidence})
}

func (h *Handler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	if err := h.service.Submit(r.Context(), actor, checkInID); err != nil {
		h.writeServiceError(w, r, "submit check-in", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submitted": true})
}

func (h *Handler) handleAddText(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	var input AddTextInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	item, err := h.service.AddTextEvidence(r.Context(), actor, checkInID, input)
	if err != nil {
		h.writeServiceError(w, r, "add text evidence", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"evidence": item})
}

func (h *Handler) handleAddLink(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	var input AddLinkInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	item, err := h.service.AddLinkEvidence(r.Context(), actor, checkInID, input)
	if err != nil {
		h.writeServiceError(w, r, "add link evidence", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"evidence": item})
}

func (h *Handler) handleAddFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds 10 MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "Multipart field 'file' is required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if _, ok := mimeToKind(mimeType); !ok {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_mime", "Unsupported file type")
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxFileSizeBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_error", "Could not read file")
		return
	}
	if int64(len(data)) > MaxFileSizeBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds 10 MB limit")
		return
	}

	item, err := h.service.AddFileEvidence(r.Context(), actor, checkInID, data, mimeType)
	if err != nil {
		h.writeServiceError(w, r, "add file evidence", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"evidence": item})
}

func (h *Handler) handleApprove(w http.ResponseWriter, r *http.Request) {
	h.handleReview(w, r, DecisionApprove)
}

func (h *Handler) handleReject(w http.ResponseWriter, r *http.Request) {
	h.handleReview(w, r, DecisionReject)
}

func (h *Handler) handleRequestChanges(w http.ResponseWriter, r *http.Request) {
	h.handleReview(w, r, DecisionRequestChanges)
}

func (h *Handler) handleReview(w http.ResponseWriter, r *http.Request, decision ReviewDecision) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}

	var input ReviewInput
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
			return
		}
	}
	input.Decision = decision

	rec, err := h.service.Review(r.Context(), actor, checkInID, input)
	if err != nil {
		h.writeServiceError(w, r, "review check-in", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"review": rec})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, ErrCheckInNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Check-in not found")
	case errors.Is(err, ErrGoalNotEligible):
		writeError(w, http.StatusForbidden, "goal_not_eligible", "Goal is not active or you are not the owner")
	case errors.Is(err, ErrNotAuthorized):
		writeError(w, http.StatusForbidden, "forbidden", "Not authorized to access this check-in")
	case errors.Is(err, ErrNotOwner):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal owner can perform this action")
	case errors.Is(err, ErrNotBuddy):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal buddy can review this check-in")
	case errors.Is(err, ErrCannotReview):
		writeError(w, http.StatusConflict, "cannot_review", "Check-in is not in a reviewable state")
	case errors.Is(err, ErrCannotSubmit):
		writeError(w, http.StatusConflict, "cannot_submit", "Check-in cannot be submitted in its current state")
	case errors.Is(err, ErrCannotAddEvidence):
		writeError(w, http.StatusConflict, "cannot_add_evidence", "Cannot add evidence in the current state")
	case errors.Is(err, ErrTooManyEvidenceItems):
		writeError(w, http.StatusConflict, "too_many_evidence", "Maximum evidence items reached")
	case errors.Is(err, ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds 10 MB limit")
	case errors.Is(err, ErrUnsupportedMIME):
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_mime", "Unsupported file type")
	case errors.Is(err, ErrInvalidEvidenceInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		if h.log != nil {
			h.log.Error(op, "err", err)
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func currentUser(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
	}
	return actor, ok
}

func pathInt64(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", param+" must be a positive integer")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
