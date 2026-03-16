package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

type csrfContextKey struct{}

// TokenFromContext returns the CSRF token stored in the request context.
func TokenFromContext(ctx context.Context) string {
	s, _ := ctx.Value(csrfContextKey{}).(string)
	return s
}

const (
	csrfCookieName = "_csrf"
	csrfTokenLen   = 32
)

// CSRF returns middleware that protects against cross-site request forgery
// using HMAC-SHA256 signed cookie tokens.
func CSRF(secret string) Middleware {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read or generate token.
			token := tokenFromCookie(r)
			if token == "" || !validSignature(token, secretBytes) {
				token = generateToken(secretBytes)
				http.SetCookie(w, &http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
			}

			// Store token in context for templates.
			ctx := context.WithValue(r.Context(), csrfContextKey{}, token)
			r = r.WithContext(ctx)

			// Validate on mutating methods.
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				headerToken := r.Header.Get("X-CSRF-Token")
				if headerToken == "" || headerToken != token {
					if r.Header.Get("HX-Request") == "true" {
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte("CSRF validation failed"))
						return
					}
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func tokenFromCookie(r *http.Request) string {
	c, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func generateToken(secret []byte) string {
	raw := make([]byte, csrfTokenLen)
	_, _ = rand.Read(raw)
	rawHex := hex.EncodeToString(raw)
	sig := sign(rawHex, secret)
	return rawHex + "." + sig
}

func sign(data string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func validSignature(token string, secret []byte) bool {
	// Token format: <hex>.<hex-signature>
	for i := range token {
		if token[i] == '.' {
			data := token[:i]
			sig := token[i+1:]
			expected := sign(data, secret)
			return hmac.Equal([]byte(sig), []byte(expected))
		}
	}
	return false
}
