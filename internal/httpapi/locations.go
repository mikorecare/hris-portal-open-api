package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/supabase-community/postgrest-go"
)

// adminListLocations lists active locations.
// @Summary      List locations
// @Tags         locations
// @Produce      json
// @Success      200  {object}  map[string]any
// @Security     ApiKeyAuth
// @Router       /locations [get]
func (h *Handlers) adminListLocations(w http.ResponseWriter, r *http.Request) {
	data, count, err := h.AdminDB.From("locations").
		Select("*", "exact", false).
		Eq("status", "active").
		Order("title", &postgrest.OrderOpts{Ascending: true}).
		Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list locations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": count,
		"data":  json.RawMessage(data),
	})
}
