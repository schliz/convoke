package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/schliz/convoke/internal/config"
	"github.com/schliz/convoke/internal/render"
	"github.com/schliz/convoke/internal/store"
	"github.com/schliz/convoke/internal/viewmodel"
)

// AppHandler is a handler that returns an error.
type AppHandler func(w http.ResponseWriter, r *http.Request) error

// Handler holds shared dependencies.
type Handler struct {
	Store    *store.Store
	Renderer *render.Renderer
	Config   *config.Config
}

// Wrap converts an AppHandler to http.HandlerFunc.
func (h *Handler) Wrap(fn AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			h.handleError(w, r, err)
		}
	}
}

// Custom error types

// NotFoundError indicates a resource was not found.
type NotFoundError struct{ Message string }

// ForbiddenError indicates the user lacks permission.
type ForbiddenError struct{ Message string }

// ValidationError indicates invalid input.
type ValidationError struct{ Message string }

func (e *NotFoundError) Error() string   { return e.Message }
func (e *ForbiddenError) Error() string  { return e.Message }
func (e *ValidationError) Error() string { return e.Message }

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var notFound *NotFoundError
	var forbidden *ForbiddenError
	var validation *ValidationError

	code := http.StatusInternalServerError
	msg := "Internal Server Error"

	switch {
	case errors.As(err, &notFound):
		code = http.StatusNotFound
		msg = notFound.Message
	case errors.As(err, &forbidden):
		code = http.StatusForbidden
		msg = forbidden.Message
	case errors.As(err, &validation):
		code = http.StatusBadRequest
		msg = validation.Message
	default:
		slog.Error("unhandled error", "error", err, "path", r.URL.Path)
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(code)
		h.Renderer.Component(w, "toast", viewmodel.Toast{
			Type: "error", Message: msg,
		})
		return
	}
	http.Error(w, msg, code)
}
