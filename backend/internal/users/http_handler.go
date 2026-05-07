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
	log               *slog.Logger
	service           *Service
	cookieName        string
	refreshCookieName string
	cookieDomain      string
	secureCookie      bool
}

// HandlerConfig is the new constructor — keeps the call sites readable now
// that the cookie story has more knobs (refresh cookie name, domain).
type HandlerConfig struct {
	Log               *slog.Logger
	Service           *Service
	CookieName        string
	RefreshCookieName string
	// CookieDomain is the Domain attribute applied to both cookies in
	// production. Empty string ⇒ omit Domain (host-only cookie). Use this in
	// production to pin cookies to the apex domain so a misrouted Next.js
	// rewrite cannot drop them.
	CookieDomain string
	SecureCookie bool
}

// NewHandler keeps the legacy positional constructor for back-compat with
// existing tests. Prefer NewHandlerWithConfig in new code.
func NewHandler(log *slog.Logger, service *Service, cookieName string, secureCookie bool) *Handler {
	return NewHandlerWithConfig(HandlerConfig{
		Log:               log,
		Service:           service,
		CookieName:        cookieName,
		RefreshCookieName: "pf_refresh",
		SecureCookie:      secureCookie,
	})
}

func NewHandlerWithConfig(cfg HandlerConfig) *Handler {
	refreshName := cfg.RefreshCookieName
	if refreshName == "" {
		refreshName = "pf_refresh"
	}
	return &Handler{
		log:               cfg.Log,
		service:           cfg.Service,
		cookieName:        cfg.CookieName,
		refreshCookieName: refreshName,
		cookieDomain:      cfg.CookieDomain,
		secureCookie:      cfg.SecureCookie,
	}
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/register", h.handleRegister)
	r.Post("/login", h.handleLogin)
	r.Post("/auth/refresh", h.handleRefresh)
	r.Post("/auth/logout", h.handleLogout)
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

	h.setAccessCookie(w, result.SessionToken, result.ExpiresAt)
	if result.RefreshTokenRaw != "" {
		h.setRefreshCookie(w, result.RefreshTokenRaw, result.RefreshExpiresAt)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": result.User,
	})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "user_not_found", "No account found for this email")
		default:
			if h.log != nil {
				h.log.Error("login user", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not create session")
		}
		return
	}

	h.setAccessCookie(w, result.SessionToken, result.ExpiresAt)
	if result.RefreshTokenRaw != "" {
		h.setRefreshCookie(w, result.RefreshTokenRaw, result.RefreshExpiresAt)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": result.User,
	})
}

// handleRefresh swaps an expired access cookie for a fresh one (and rotates
// the refresh cookie). The frontend calls this on 401 transparently — see
// web/lib/api.ts apiFetch wrapper.
func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "auth_required", "No refresh cookie")
		return
	}

	result, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, ErrRefreshTokenInvalid), errors.Is(err, ErrRefreshTokenReused):
			// Reuse and «invalid» both return 401 to the client. Reuse already
			// triggered chain revocation server-side; either way the client must
			// log in again. We also clear the refresh cookie so the SPA stops
			// retrying with a value the server has blacklisted.
			h.clearRefreshCookie(w)
			writeError(w, http.StatusUnauthorized, "invalid_session", "Refresh token is invalid")
		default:
			if h.log != nil {
				h.log.Error("refresh session", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not refresh session")
		}
		return
	}

	h.setAccessCookie(w, result.SessionToken, result.ExpiresAt)
	h.setRefreshCookie(w, result.RefreshTokenRaw, result.RefreshExpiresAt)

	writeJSON(w, http.StatusOK, map[string]any{
		"user": result.User,
	})
}

// handleLogout deletes both cookies and best-effort revokes them on the
// server. Always 204 — the user is effectively logged out either way.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var rawAccess, rawRefresh string
	if c, err := r.Cookie(h.cookieName); err == nil {
		rawAccess = c.Value
	}
	if c, err := r.Cookie(h.refreshCookieName); err == nil {
		rawRefresh = c.Value
	}

	if err := h.service.Logout(r.Context(), rawAccess, rawRefresh); err != nil && h.log != nil {
		// Logging only — we still drop the cookies so the user-visible result
		// is "logged out" even if a DB hiccup keeps the row alive briefly.
		h.log.Warn("logout: server-side revoke failed", "err", err)
	}

	h.clearAccessCookie(w)
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
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

// ── Cookie helpers ───────────────────────────────────────────────────────────
//
// Two cookies, two scopes:
//   * pf_session  (access)  — Path=/, sent with every API request.
//   * pf_refresh  (refresh) — Path=/v1/auth, only sent when refreshing or
//                              logging out, so it isn't on the wire for normal
//                              traffic. Smaller exposure window if a
//                              same-origin XSS leaks one but not the other.

func (h *Handler) setAccessCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    value,
		Path:     "/",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshCookieName,
		Value:    value,
		Path:     "/v1/auth",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func (h *Handler) clearAccessCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshCookieName,
		Value:    "",
		Path:     "/v1/auth",
		Domain:   h.cookieDomain,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
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
