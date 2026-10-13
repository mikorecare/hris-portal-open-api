package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/supabase-community/postgrest-go"
)

// ---------- Types ----------

// TeamMember is the public shape emitted by the API.
// NOTE: Role is populated from user_profiles.title, NOT the DB role column.
type TeamMember struct {
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	AboutMe     *string `json:"about_me"`
	Role        string  `json:"role,omitempty"`
}

// TeamMemberRow is the raw DB shape returned by Supabase.
// Pointer fields because Postgres columns are nullable.
type TeamMemberRow struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	AboutMe     *string `json:"about_me"`
	Title       *string `json:"title"`
}

// ---------- Handlers ----------

// listJobs lists published jobs with optional filters.
// @Summary      List jobs
// @Description  Returns published, active, non-private jobs.
// @Tags         jobs
// @Produce      json
// @Param        employment_type  query  string  false  "Filter by employment type"
// @Param        city             query  string  false  "Filter by city"
// @Param        limit            query  int     false  "Max results (default 20, max 100)"
// @Success      200  {object}  map[string]any
// @Failure      401  {string}  string  "Missing or invalid API key"
// @Failure      500  {string}  string  "Internal error"
// @Security     ApiKeyAuth
// @Router       /jobs [get]
func (h *Handlers) listJobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := h.DB.From("jobs").
		Select("id,title,description,employment_type,department,country,state,city,is_telecommute,public_minimum_salary,public_maximum_salary,banner_image_url,thumbnail_image_url,published_at,team_ids", "exact", false).
		Eq("status", "published").
		Eq("is_active", "true").
		Eq("is_private", "false").
		Eq("is_anonymous", "false")

	if v := q.Get("employment_type"); v != "" {
		query = query.Eq("employment_type", v)
	}
	if v := q.Get("department"); v != "" {
		query = query.Eq("department", v)
	}
	if v := q.Get("country"); v != "" {
		query = query.Eq("country", v)
	}
	if v := q.Get("state"); v != "" {
		query = query.Eq("state", v)
	}
	if v := q.Get("city"); v != "" {
		query = query.Eq("city", v)
	}
	if v := q.Get("is_telecommute"); v == "true" || v == "false" {
		query = query.Eq("is_telecommute", v)
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		query = query.Ilike("title", "%"+v+"%")
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
		Order("published_at", &postgrest.OrderOpts{Ascending: false}).
		Range(offset, offset+limit-1, "").
		Execute()
	if err != nil {
		log.Printf("listJobs: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	// Parse into maps so we can rewrite team_ids -> team_members.
	var jobs []map[string]any
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("listJobs: unmarshal failed. raw=%s err=%v", string(data), err)
		writeError(w, http.StatusInternalServerError, "failed to parse jobs")
		return
	}

	// Collect every team ID across all jobs (deduped).
	idSet := map[string]struct{}{}
	for _, job := range jobs {
		for _, id := range extractTeamIDs(job["team_ids"]) {
			idSet[id] = struct{}{}
		}
	}

	// One batch query for all members.
	members, err := h.fetchTeamMembers(keys(idSet))
	if err != nil {
		log.Printf("listJobs: fetchTeamMembers failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch team members")
		return
	}

	// Index rows by ID.
	rowByID := make(map[string]TeamMemberRow, len(members))
	for _, m := range members {
		rowByID[m.ID] = m
	}

	// Attach team_members to each job and drop the raw team_ids.
	for _, job := range jobs {
		ids := extractTeamIDs(job["team_ids"])
		delete(job, "team_ids")

		attached := make([]TeamMember, 0, len(ids))
		for _, id := range ids {
			row, ok := rowByID[id]
			if !ok {
				continue
			}
			attached = append(attached, toTeamMember(row))
		}
		job["team_members"] = attached
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":  count,
		"limit":  limit,
		"offset": offset,
		"data":   jobs,
	})
}

// getJob handles GET /jobs/{id}, embedding team_members.
func (h *Handlers) getJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing job id")
		return
	}

	data, _, err := h.DB.From("jobs").
		Select("id,title,description,employment_type,minimum_experience,department,country,state,city,street_address,postal_code,is_telecommute,lat,lng,public_minimum_salary,public_maximum_salary,banner_image_url,thumbnail_image_url,banner_image_alt,thumbnail_image_alt,application_questions,meta_description,job_reference,published_at,team_ids", "exact", false).
		Eq("id", id).
		Eq("status", "published").
		Eq("is_active", "true").
		Eq("is_private", "false").
		Eq("is_anonymous", "false").
		Execute()
	if err != nil {
		log.Printf("getJob: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}

	if string(data) == "[]" {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil || len(rows) == 0 {
		log.Printf("getJob: unmarshal failed. raw=%s err=%v", string(data), err)
		writeError(w, http.StatusInternalServerError, "failed to parse job")
		return
	}
	job := rows[0]

	ids := extractTeamIDs(job["team_ids"])
	delete(job, "team_ids")

	members, err := h.fetchTeamMembers(ids)
	if err != nil {
		log.Printf("getJob: fetchTeamMembers failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch team members")
		return
	}

	rowByID := make(map[string]TeamMemberRow, len(members))
	for _, m := range members {
		rowByID[m.ID] = m
	}

	attached := make([]TeamMember, 0, len(ids))
	for _, id := range ids {
		row, ok := rowByID[id]
		if !ok {
			continue
		}
		attached = append(attached, toTeamMember(row))
	}
	job["team_members"] = attached

	writeJSON(w, http.StatusOK, job)
}

// ---------- Helpers ----------

// fetchTeamMembers loads user_profiles rows for the given IDs.
// Uses the admin client because user_profiles is RLS-protected.
func (h *Handlers) fetchTeamMembers(ids []string) ([]TeamMemberRow, error) {
	out := []TeamMemberRow{}
	if len(ids) == 0 {
		return out, nil
	}

	// PostgREST "in" filter: id=in.(uuid1,uuid2,...)
	inList := "(" + strings.Join(ids, ",") + ")"

	data, _, err := h.AdminDB.From("user_profiles").
		Select("id,display_name,avatar_url,about_me,title", "exact", false).
		Filter("id", "in", inList).
		Execute()
	if err != nil {
		log.Printf("fetchTeamMembers: query failed: %v", err)
		return nil, err
	}

	if err := json.Unmarshal(data, &out); err != nil {
		log.Printf("fetchTeamMembers: unmarshal failed. raw=%s err=%v", string(data), err)
		return nil, err
	}
	return out, nil
}

// toTeamMember converts a raw DB row into the public TeamMember shape.
// Mirrors the Next.js mapping: `role` on the wire is filled from DB `title`.
func toTeamMember(row TeamMemberRow) TeamMember {
	title := ""
	if row.Title != nil {
		title = *row.Title
	}
	return TeamMember{
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		AboutMe:     row.AboutMe,
		Role:        title,
	}
}

// extractTeamIDs normalizes a Supabase-returned uuid[] value into []string.
func extractTeamIDs(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// keys returns the map keys as a sorted slice for deterministic URLs.
func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
