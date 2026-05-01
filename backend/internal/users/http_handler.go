package users

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	log          *slog.Logger
	service      *Service
	cookieName   string
	secureCookie bool
}

func NewHandler(log *slog.Logger, service *Service, cookieName string, secureCookie bool) *Handler {
	return &Handler{
		log:          log,
		service:      service,
		cookieName:   cookieName,
		secureCookie: secureCookie,
	}
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/register", h.handleRegister)
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/me", h.handleMe)
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.cookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
			return
		}

		user, err := h.service.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, "invalid_session", "Session is invalid or expired")
				return
			}

			if h.log != nil {
				h.log.Error("authenticate request", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not authenticate request")
			return
		}

		next.ServeHTTP(w, r.WithContext(WithAuthenticatedUser(r.Context(), user)))
	})
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	result, err := h.service.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		case errors.Is(err, ErrEmailTaken):
			writeError(w, http.StatusConflict, "email_taken", "User with this email already exists")
		default:
			if h.log != nil {
				h.log.Error("register user", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not create account")
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    result.SessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
		Expires:  result.ExpiresAt,
		MaxAge:   int(time.Until(result.ExpiresAt).Seconds()),
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": result.User,
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": user,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
