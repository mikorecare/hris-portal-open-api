package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseServiceKey string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		SupabaseURL:        os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:    os.Getenv("SUPABASE_ANON_KEY"),
		SupabaseServiceKey: os.Getenv("SUPABASE_SERVICE_KEY"),
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseAnonKey == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_ANON_KEY must be set")
	}
	if cfg.SupabaseServiceKey == "" {
		log.Fatal("SUPABASE_SERVICE_KEY must be set")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
