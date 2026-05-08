package authz

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

func writeJSON403(w http.ResponseWriter) {
	http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
}

// RequirePlatformAdmin returns a middleware that returns 403 if the
// authenticated user does not have the platform_admin flag.
func RequirePlatformAdmin(az *Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := users.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// Fast path: the field is already in-memory from the session lookup.
			if !actor.IsPlatformAdmin {
				writeJSON403(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireWorkspaceOwner returns a middleware that returns 403 if the
// authenticated user is not the owner of the workspace identified by
// the {workspaceID} URL parameter.
func RequireWorkspaceOwner(az *Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := users.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			workspaceID, err := strconv.ParseInt(chi.URLParam(r, "workspaceID"), 10, 64)
			if err != nil {
				http.Error(w, "invalid workspace id", http.StatusBadRequest)
				return
			}
			isOwner, err := az.IsWorkspaceOwner(r.Context(), actor.ID, workspaceID)
			if err != nil || !isOwner {
				writeJSON403(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireTeamspaceLead returns a middleware that returns 403 if the
// authenticated user is not a lead in the team identified by the
// {teamID} URL parameter.
func RequireTeamspaceLead(az *Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := users.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			teamID, err := strconv.ParseInt(chi.URLParam(r, "teamID"), 10, 64)
			if err != nil {
				http.Error(w, "invalid team id", http.StatusBadRequest)
				return
			}
			isLead, err := az.IsTeamspaceLead(r.Context(), actor.ID, teamID)
			if err != nil || !isLead {
				writeJSON403(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
