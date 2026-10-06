package httpapi

import (
	"net/http"

	"recruiting-api/internal/supabase"
)

func NewRouter(db, adminDB *supabase.Client) (open, protected *http.ServeMux) {
	h := &Handlers{DB: db, AdminDB: adminDB}

	const base = "/api/public"

	open = http.NewServeMux()
	open.HandleFunc("GET "+base, h.Root)             // exact
	open.HandleFunc("GET "+base+"/health", h.Health) // exact

	protected = http.NewServeMux()
	protected.HandleFunc("GET "+base+"/jobs", h.listJobs)
	protected.HandleFunc("GET "+base+"/jobs/{id}", h.getJob)
	protected.HandleFunc("POST "+base+"/apply", h.apply)
	protected.HandleFunc("GET "+base+"/locations", h.adminListLocations)

	return open, protected
}
