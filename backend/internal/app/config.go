package app

import (
	"os"
	"strings"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	GeminiAPIKey string
	GeminiModel  string
	CORSOrigin   string
}

func LoadConfig() Config {
	loadDotEnv()
	return Config{
		Port:         env("PORT", "8080"),
		DatabaseURL:  env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/campus_market?sslmode=disable"),
		JWTSecret:    env("JWT_SECRET", "dev-secret-change-me"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  env("GEMINI_MODEL", "gemini-2.5-flash-lite"),
		CORSOrigin:   env("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func loadDotEnv() {
	for _, path := range []string{".env", "backend/.env"} {
		data, err := os.ReadFile(path)
		if err == nil {
			applyDotEnv(string(data))
			return
		}
	}
}

func applyDotEnv(data string) {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
