package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/supabase-community/postgrest-go"
)

// adminListJobTitles lists active job titles.
// @Summary      List job titles
// @Tags         job-titles
// @Produce      json
// @Success      200  {object}  map[string]any
// @Security     ApiKeyAuth
// @Router       /job-titles [get]
func (h *Handlers) adminListJobTitles(w http.ResponseWriter, r *http.Request) {
	data, count, err := h.AdminDB.From("settings_people_job_titles").
		Select("*", "exact", false).
		Eq("status", "active").
		Order("name", &postgrest.OrderOpts{Ascending: true}).
		Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list job titles")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": count,
		"data":  json.RawMessage(data),
	})
}
