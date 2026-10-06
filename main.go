package main

import (
	"log"
	"net/http"
	"time"

	"recruiting-api/internal/apikeys"
	"recruiting-api/internal/config"
	"recruiting-api/internal/httpapi"
	"recruiting-api/internal/middleware"
	"recruiting-api/internal/supabase"
)

func main() {
	//refire the deploy
	cfg := config.Load()

	publicDB := supabase.NewPublic(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	adminDB := supabase.NewAdmin(cfg.SupabaseURL, cfg.SupabaseServiceKey)

	openMux, protectedMux := httpapi.NewRouter(publicDB, adminDB)
	apiKeyCache := apikeys.NewCache(30 * time.Second)
	apiKeyAuth := middleware.NewAPIKeyAuth(adminDB, apiKeyCache)

	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		for range t.C {
			apiKeyCache.Cleanup()
		}
	}()

	limiter := middleware.NewIPLimiter(10, 20)
	protectedHandler := limiter.Middleware(apiKeyAuth.Middleware(protectedMux))
	openHandler := limiter.Middleware(openMux)
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := openMux.Handler(r); pattern != "" {
			openHandler.ServeHTTP(w, r)
			return
		}
		protectedHandler.ServeHTTP(w, r)
	})

	handler := middleware.CORS(root)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}
