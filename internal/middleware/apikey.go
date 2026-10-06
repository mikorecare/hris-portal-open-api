package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"recruiting-api/internal/apikeys"
	"recruiting-api/internal/supabase"
)

// ContextKey is the type for request-scoped values set by this middleware.
type ContextKey string

const APIKeyUserIDKey ContextKey = "apikey_user_id"
const APIKeyTypeKey ContextKey = "apikey_type"

// APIKeyAuth authenticates requests against the api_keys table.
type APIKeyAuth struct {
	DB    *supabase.Client
	Cache *apikeys.Cache
}

func NewAPIKeyAuth(db *supabase.Client, cache *apikeys.Cache) *APIKeyAuth {
	return &APIKeyAuth{DB: db, Cache: cache}
}

func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key == "" {
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		if e, ok := a.Cache.Get(key); ok {
			if e.Revoked || e.KeyType != "api" {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), APIKeyUserIDKey, e.UserID)
			ctx = context.WithValue(ctx, APIKeyTypeKey, e.KeyType)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Cache miss — DB lookup.
		row, err := a.lookup(key)
		if err != nil {
			log.Printf("apikey: lookup failed: %v", err)
			http.Error(w, "auth service unavailable", http.StatusServiceUnavailable)
			return
		}

		if row == nil || row.revoked() || row.KeyType != "api" {
			// Negative-cache to prevent hammering the DB with bad keys.
			a.Cache.Set(key, "", "", true)
			http.Error(w, "invalid API key", http.StatusUnauthorized)
			return
		}

		a.Cache.Set(key, row.KeyType, row.UserID, false)

		ctx := context.WithValue(r.Context(), APIKeyUserIDKey, row.UserID)
		ctx = context.WithValue(ctx, APIKeyTypeKey, row.KeyType)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type apiKeyRow struct {
	KeyType   string  `json:"key_type"`
	UserID    string  `json:"user_id"`
	RevokedAt *string `json:"revoked_at"`
}

func (r apiKeyRow) revoked() bool {
	return r.RevokedAt != nil && *r.RevokedAt != ""
}

func (a *APIKeyAuth) lookup(key string) (*apiKeyRow, error) {
	data, _, err := a.DB.From("api_keys").
		Select("key_type,user_id,revoked_at", "exact", false).
		Eq("key_value", key).
		Execute()
	if err != nil {
		return nil, err
	}

	var rows []apiKeyRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func extractAPIKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}
