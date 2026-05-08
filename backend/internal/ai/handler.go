package ai

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes all AI assistant endpoints.
type Handler struct {
	provider  AssistantProvider
	goalRepo  GoalReader
	proofRepo ProofReader
	tracker   analytics.Tracker
}

// GoalReader is the minimal interface needed by the AI handler to fetch goal context.
type GoalReader interface {
	GetGoalText(ctx context.Context, goalID int64, userID int64) (string, error)
	GetRecentActivityText(ctx context.Context, userID int64) (string, error)
}

// ProofReader fetches proof texts for dossier generation.
type ProofReader interface {
	GetProofTextsForDossier(ctx context.Context, userID int64, startDate, endDate string) ([]DossierInput, error)
}

func NewHandler(provider AssistantProvider, goalRepo GoalReader, proofRepo ProofReader, tracker analytics.Tracker) *Handler {
	return &Handler{provider: provider, goalRepo: goalRepo, proofRepo: proofRepo, tracker: tracker}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/ai/goal-to-proofs", h.handleGoalToProofs)
	r.Post("/ai/next-step", h.handleNextStep)
	r.Post("/ai/proof-check", h.handleProofCheck)
	r.Post("/ai/anti-proof", h.handleAntiProof)
	r.Post("/ai/growth-dossier", h.handleGrowthDossier)
}

// POST /v1/ai/goal-to-proofs
func (h *Handler) handleGoalToProofs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		GoalText string `json:"goal_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.GoalText == "" {
		writeErrJSON(w, http.StatusBadRequest, "invalid_input", "goal_text required")
		return
	}

	result, err := h.provider.GoalToProofs(r.Context(), in.GoalText)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "ai_error", "Could not generate suggestions")
		return
	}

	if actor, ok := users.CurrentUser(r.Context()); ok {
		h.tracker.TrackAsync(analytics.PilotEvent{
			UserID: actor.ID,
			Name:   analytics.EventAiSuggestionUsed,
			Props:  map[string]any{"kind": "goal_to_proofs"},
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// POST /v1/ai/next-step
func (h *Handler) handleNextStep(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	var in struct {
		GoalID int64 `json:"goal_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.GoalID == 0 {
		writeErrJSON(w, http.StatusBadRequest, "invalid_input", "goal_id required")
		return
	}

	goalText, err := h.goalRepo.GetGoalText(r.Context(), in.GoalID, actor.ID)
	if err != nil {
		writeErrJSON(w, http.StatusNotFound, "goal_not_found", "Goal not found")
		return
	}

	activity, _ := h.goalRepo.GetRecentActivityText(r.Context(), actor.ID)

	result, err := h.provider.NextStep(r.Context(), goalText, activity)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "ai_error", "Could not generate next step")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /v1/ai/proof-check
func (h *Handler) handleProofCheck(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Draft string `json:"draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Draft == "" {
		writeErrJSON(w, http.StatusBadRequest, "invalid_input", "draft required")
		return
	}

	result, err := h.provider.ProofCheck(r.Context(), in.Draft)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "ai_error", "Could not check proof")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /v1/ai/anti-proof
func (h *Handler) handleAntiProof(w http.ResponseWriter, r *http.Request) {
	var in struct {
		GoalText         string `json:"goal_text"`
		StuckDescription string `json:"stuck_description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.GoalText == "" {
		writeErrJSON(w, http.StatusBadRequest, "invalid_input", "goal_text and stuck_description required")
		return
	}

	result, err := h.provider.AntiProof(r.Context(), in.GoalText, in.StuckDescription)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "ai_error", "Could not generate anti-proof")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /v1/ai/growth-dossier
func (h *Handler) handleGrowthDossier(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	var in struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.StartDate == "" || in.EndDate == "" {
		writeErrJSON(w, http.StatusBadRequest, "invalid_input", "start_date and end_date required")
		return
	}

	inputs, err := h.proofRepo.GetProofTextsForDossier(r.Context(), actor.ID, in.StartDate, in.EndDate)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "db_error", "Could not fetch proof data")
		return
	}

	if len(inputs) == 0 {
		writeErrJSON(w, http.StatusUnprocessableEntity, "no_data", "No proofs found for the given period")
		return
	}

	result, err := h.provider.GrowthDossier(r.Context(), inputs)
	if err != nil {
		writeErrJSON(w, http.StatusInternalServerError, "ai_error", "Could not generate dossier")
		return
	}

	h.tracker.TrackAsync(analytics.PilotEvent{
		UserID: actor.ID,
		Name:   analytics.EventDossierGenerated,
		Props:  map[string]any{"start_date": in.StartDate, "end_date": in.EndDate},
	})

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrJSON(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": code, "message": message})
}
