package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/supabase-community/postgrest-go"
)

// listPeople lists active, public-facing team members.
// @Summary      List people
// @Tags         people
// @Produce      json
// @Security     ApiKeyAuth
// @Router       /people [get]
func (h *Handlers) listPeople(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := h.AdminDB.From("people").
		Select("id,full_name,first_name,last_name,job_title,department_id,bio,display_photo_path,location_type,country,work_email", "exact", false).
		Eq("is_approved", "true").
		Eq("is_displayed_on_web", "true").
		Eq("employment_status", "active")

	if v := q.Get("department_id"); v != "" {
		query = query.Eq("department_id", v)
	}
	if v := q.Get("location_type"); v != "" {
		query = query.Eq("location_type", v)
	}
	if v := q.Get("country"); v != "" {
		query = query.Eq("country", v)
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		search := "%" + v + "%"
		condition := "full_name.ilike." + search + ",job_title.ilike." + search
		query = query.Or(condition, "")
	}

	limit := 20
	offset := 0
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = v
	}

	data, count, err := query.
		Order("full_name", &postgrest.OrderOpts{Ascending: true}).
		Range(offset, offset+limit-1, "").
		Execute()
	if err != nil {
		log.Printf("listPeople: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list people")
		return
	}

	var people []map[string]any
	if err := json.Unmarshal(data, &people); err != nil {
		log.Printf("listPeople: unmarshal failed. raw=%s err=%v", string(data), err)
		writeError(w, http.StatusInternalServerError, "failed to parse people")
		return
	}

	// Pull the base URL from environment variables
	supabaseURL := os.Getenv("SUPABASE_URL")
	baseURL := supabaseURL + "/storage/v1/object/public/people-photos/"

	for _, person := range people {
		path, ok := person["display_photo_path"].(string)
		if ok && path != "" {
			person["display_photo_url"] = baseURL + path
		} else {
			person["display_photo_url"] = nil
		}
		delete(person, "display_photo_path")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": count,
		"data":  people,
	})
}

// getPerson handles GET /people/{id}.
// @Summary      Get person
// @Tags         people
// @Produce      json
// @Security     ApiKeyAuth
// @Router       /people/{id} [get]
func (h *Handlers) getPerson(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing person id")
		return
	}

	data, _, err := h.AdminDB.From("people").
		Select("id,full_name,first_name,last_name,job_title,department_id,bio,display_photo_path,location_type,country,work_email", "exact", false).
		Eq("id", id).
		Eq("is_approved", "true").
		Eq("is_displayed_on_web", "true").
		Eq("employment_status", "active").
		Execute()
	if err != nil {
		log.Printf("getPerson: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch person")
		return
	}

	if string(data) == "[]" {
		writeError(w, http.StatusNotFound, "person not found or not public")
		return
	}

	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil || len(rows) == 0 {
		log.Printf("getPerson: unmarshal failed. raw=%s err=%v", string(data), err)
		writeError(w, http.StatusInternalServerError, "failed to parse person")
		return
	}

	person := rows[0]

	// Pull the base URL from environment variables
	supabaseURL := os.Getenv("SUPABASE_URL")
	baseURL := supabaseURL + "/storage/v1/object/public/people-photos/"

	path, ok := person["display_photo_path"].(string)
	if ok && path != "" {
		person["display_photo_url"] = baseURL + path
	} else {
		person["display_photo_url"] = nil
	}
	delete(person, "display_photo_path")

	writeJSON(w, http.StatusOK, person)
}
