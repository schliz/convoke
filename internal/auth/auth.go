package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"

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

// Middleware extracts user from X-Forwarded-Email/Groups headers.
// In dev mode, injects a fake admin user when no headers are present.
func Middleware(s *store.Store, adminGroup string, devMode bool) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email := r.Header.Get("X-Forwarded-Email")

			if email == "" && devMode {
				// Dev bypass: inject fake admin user
				ctx := context.WithValue(r.Context(), contextKey{}, &RequestUser{
					ID:      0,
					Email:   "dev@localhost",
					IsAdmin: true,
					Groups:  []string{"admin"},
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if email == "" {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			groups := parseGroups(r.Header.Get("X-Forwarded-Groups"))
			isAdmin := slices.Contains(groups, adminGroup)
			displayName := r.Header.Get("X-Forwarded-Preferred-Username")
			if displayName == "" {
				displayName = email
			}

			user, err := store.GetOrCreateUser(r.Context(), s.DB(), email, displayName, groups, isAdmin)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &RequestUser{
				ID:      user.ID,
				Email:   user.Email,
				IsAdmin: user.IsAdmin,
				Groups:  user.Groups,
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
