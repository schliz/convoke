package auth

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/schliz/convoke/internal/db"
	"github.com/schliz/convoke/internal/middleware"
	"github.com/schliz/convoke/internal/store"
)

type RequestUser struct {
	ID      int64
	Email   string
	IsAdmin bool
	Groups  []string
}

type contextKey struct{}

func UserFromContext(ctx context.Context) *RequestUser {
	u, _ := ctx.Value(contextKey{}).(*RequestUser)
	return u
}

// ContextWithUser stores a RequestUser in the context. This is exported for
// use in tests and in handler helpers that need to inject a user.
func ContextWithUser(ctx context.Context, user *RequestUser) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// Middleware extracts user identity from oauth2-proxy headers, upserts
// the user into the database via sqlc, and stores a RequestUser in the
// request context.
//
// Header mapping (set by oauth2-proxy / mock-oauth2-proxy):
//
//	X-Auth-Request-User               -> idp_subject (OIDC sub claim)
//	X-Auth-Request-Email              -> email
//	X-Auth-Request-Preferred-Username -> username
//	X-Auth-Request-Groups             -> groups (comma-separated)
//
// display_name is derived from username, falling back to the email prefix.
// is_assoc_admin is determined by checking if adminGroup is in the groups
// list BEFORE calling UpsertUser.
func Middleware(s *store.Store, adminGroup string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idpSubject := r.Header.Get("X-Auth-Request-User")

			if idpSubject == "" {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			email := r.Header.Get("X-Auth-Request-Email")
			username := r.Header.Get("X-Auth-Request-Preferred-Username")
			groups := parseGroups(r.Header.Get("X-Auth-Request-Groups"))

			// Derive display_name from username, falling back to email prefix.
			displayName := username
			if displayName == "" {
				if idx := strings.Index(email, "@"); idx > 0 {
					displayName = email[:idx]
				} else {
					displayName = idpSubject
				}
			}

			// If no username header, use display_name as username.
			if username == "" {
				username = displayName
			}

			// Determine admin status from groups before upserting.
			isAdmin := slices.Contains(groups, adminGroup)

			// Upsert user and sync IdP groups.
			user, err := s.Queries().UpsertUser(r.Context(), db.UpsertUserParams{
				IdpSubject:   idpSubject,
				Username:     username,
				DisplayName:  displayName,
				Email:        email,
				IsAssocAdmin: isAdmin,
			})
			if err != nil {
				slog.Error("auth: failed to upsert user", "error", err, "idp_subject", idpSubject)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Sync IdP groups: delete all then re-insert.
			if err := s.Queries().DeleteUserIDPGroups(r.Context(), user.ID); err != nil {
				slog.Error("auth: failed to delete user idp groups", "error", err, "user_id", user.ID)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			for _, g := range groups {
				if err := s.Queries().InsertUserIDPGroup(r.Context(), db.InsertUserIDPGroupParams{
					UserID:    user.ID,
					GroupName: g,
				}); err != nil {
					slog.Error("auth: failed to insert user idp group", "error", err, "user_id", user.ID, "group", g)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &RequestUser{
				ID:      user.ID,
				Email:   user.Email,
				IsAdmin: isAdmin,
				Groups:  groups,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseGroups(header string) []string {
	if header == "" {
		return nil
	}
	var groups []string
	for g := range strings.SplitSeq(header, ",") {
		if g = strings.TrimSpace(g); g != "" {
			groups = append(groups, g)
		}
	}
	return groups
}
