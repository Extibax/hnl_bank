package config

import "os"

// Config holds runtime configuration sourced from environment variables.
type Config struct {
	Port             string
	PostgresDSN      string
	TigerBeetleAddr  string
	JWTSecret        string
	OpenRouterAPIKey string
	OpenRouterModel  string
}

// Load reads configuration from the environment, falling back to defaults.
func Load() Config {
	return Config{
		Port:             getenv("PORT", "8080"),
		PostgresDSN:      getenv("DATABASE_URL", "postgres://hnlbank:hnlbank@localhost:5432/hnlbank?sslmode=disable"),
		TigerBeetleAddr:  getenv("TIGERBEETLE_ADDRESS", "3000"),
		JWTSecret:        getenv("JWT_SECRET", "dev-secret-change-me"),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:  getenv("OPENROUTER_MODEL", "deepseek/deepseek-chat-v3-0324"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
