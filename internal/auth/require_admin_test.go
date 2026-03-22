package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAdmin(inner)

	req := httptest.NewRequest("GET", "/admin/units/", nil)
	ctx := context.WithValue(req.Context(), contextKey{}, &RequestUser{
		ID: 1, Email: "admin@example.com", IsAdmin: true, Groups: []string{"admin"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected inner handler to be called for admin user")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestRequireAdmin_BlocksNonAdmin(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireAdmin(inner)

	req := httptest.NewRequest("GET", "/admin/units/", nil)
	ctx := context.WithValue(req.Context(), contextKey{}, &RequestUser{
		ID: 2, Email: "user@example.com", IsAdmin: false, Groups: []string{"members"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected inner handler NOT to be called for non-admin user")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}
}

func TestRequireAdmin_BlocksNonAdminHTMX(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireAdmin(inner)

	req := httptest.NewRequest("GET", "/admin/units/", nil)
	req.Header.Set("HX-Request", "true")
	ctx := context.WithValue(req.Context(), contextKey{}, &RequestUser{
		ID: 2, Email: "user@example.com", IsAdmin: false, Groups: []string{"members"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected inner handler NOT to be called for non-admin HTMX request")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}
}

func TestRequireAdmin_BlocksNoUser(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireAdmin(inner)

	req := httptest.NewRequest("GET", "/admin/units/", nil)
	// No user in context

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected inner handler NOT to be called when no user in context")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}
}
