package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/supabase-community/postgrest-go"
)

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
