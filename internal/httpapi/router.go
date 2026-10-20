package httpapi

import (
	"net/http"

	_ "recruiting-api/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"recruiting-api/internal/supabase"
)

func NewRouter(db, adminDB *supabase.Client) (open, protected *http.ServeMux) {
	h := &Handlers{DB: db, AdminDB: adminDB}

	const base = "/api/public"

	open = http.NewServeMux()
	open.HandleFunc("GET "+base, h.Root)
	open.HandleFunc("GET "+base+"/health", h.Health)
	open.Handle("GET "+base+"/swagger/", httpSwagger.WrapHandler)

	protected = http.NewServeMux()

	// Jobs
	protected.HandleFunc("GET "+base+"/jobs", h.listJobs)
	protected.HandleFunc("GET "+base+"/jobs/{id}", h.getJob)

	// People
	protected.HandleFunc("GET "+base+"/people", h.listPeople)
	protected.HandleFunc("GET "+base+"/people/{id}", h.getPerson)

	// Other
	protected.HandleFunc("POST "+base+"/apply", h.apply)
	protected.HandleFunc("GET "+base+"/locations", h.adminListLocations)
	protected.HandleFunc("GET "+base+"/job-titles", h.adminListJobTitles)

	return open, protected
}
