package handler

import (
	"fmt"
	"net/http"
)

// HealthCheck verifies database connectivity and returns status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	err := h.Store.DB().QueryRow(r.Context(), "SELECT 1").Scan(new(int))
	if err != nil {
		return fmt.Errorf("health check: db ping: %w", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
	return nil
}
