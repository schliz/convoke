package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schliz/convoke/internal/auth"
)

func TestValidateSlug_ValidSlugs(t *testing.T) {
	tests := []struct {
		slug string
		ok   bool
	}{
		{"fire-brigade", true},
		{"alpha", true},
		{"test-123", true},
		{"a-b-c", true},
		{"abc123", true},
		{"", false},
		{"Fire-Brigade", false},   // uppercase
		{"fire_brigade", false},   // underscore
		{"fire brigade", false},   // space
		{"-leading-dash", false},  // leading dash
		{"trailing-dash-", false}, // trailing dash
		{"double--dash", false},   // double dash
		{"a", true},               // single char
		{"123", true},             // digits only
		{"a-1", true},             // mixed
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := isValidSlug(tt.slug)
			if got != tt.ok {
				t.Errorf("isValidSlug(%q) = %v, want %v", tt.slug, got, tt.ok)
			}
		})
	}
}

func TestParseGroupBindings_FiltersEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"normal groups", []string{"group-a", "group-b"}, []string{"group-a", "group-b"}},
		{"with empty strings", []string{"group-a", "", "group-b", ""}, []string{"group-a", "group-b"}},
		{"all empty", []string{"", "", ""}, nil},
		{"nil input", nil, nil},
		{"with whitespace", []string{" group-a ", "  ", "group-b"}, []string{"group-a", "group-b"}},
		{"deduplication", []string{"group-a", "group-b", "group-a"}, []string{"group-a", "group-b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGroupBindings(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseGroupBindings() returned %d items, want %d: %v", len(got), len(tt.expected), got)
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("parseGroupBindings()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestAdminCreateUnit_MissingName_ReturnsValidationError(t *testing.T) {
	// Validation logic for the handler is tested via the exported utility
	// functions (isValidSlug, parseGroupBindings). Full handler integration
	// requires a store and renderer, which is covered by e2e tests.
	t.Skip("requires store and renderer; covered by e2e tests and utility function tests")
}

func TestAdminDeleteUnit_NonHTMX_Redirects(t *testing.T) {
	t.Skip("requires store mock; covered by e2e tests")
}

func TestAdminDeleteUnit_HTMX_ReturnsEmptyBody(t *testing.T) {
	t.Skip("requires store mock; covered by e2e tests")
}

func TestRequireAdmin_ViaHTTP_BlocksNonAdmin(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := auth.RequireAdmin(inner)

	req := httptest.NewRequest("GET", "/admin/units/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("expected handler not to be called without admin user")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
