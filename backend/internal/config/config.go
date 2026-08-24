package config

import "os"

// Config holds runtime configuration sourced from environment variables.
type Config struct {
	Port             string
	PostgresDSN      string
	TigerBeetleAddr  string
	JWTSecret    string
	AIBaseURL    string
	AIAPIKey     string
	AIModel      string
}

// Load reads configuration from the environment, falling back to defaults.
func Load() Config {
	return Config{
		Port:             getenv("PORT", "8080"),
		PostgresDSN:      getenv("DATABASE_URL", "postgres://hnlbank:hnlbank@localhost:5432/hnlbank?sslmode=disable"),
		TigerBeetleAddr:  getenv("TIGERBEETLE_ADDRESS", "3000"),
		JWTSecret:   getenv("JWT_SECRET", "dev-secret-change-me"),
		AIBaseURL:   getenv("OPENAI_BASE_URL", "https://opencode.ai/zen/v1"),
		AIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		AIModel:     getenv("OPENAI_MODEL", "nemotron-3-ultra-free"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}