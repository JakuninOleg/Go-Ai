package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type APIConfig struct {
	APIKey  string
	BaseURL string
}

type Config struct {
	Port                   string
	SharedSecret           string
	ModelRefreshInterval   time.Duration
	GroqSTTMaxRequestBytes int64

	Providers struct {
		Gemini     APIConfig
		Groq       APIConfig
		OpenRouter APIConfig
	}
}

func Load() Config {
	godotenv.Load()

	return Config{
		Port:                   getEnv("PORT", "8080"),
		SharedSecret:           os.Getenv("GO_AI_SHARED_SECRET"),
		ModelRefreshInterval:   getDurationEnv("MODEL_REFRESH_INTERVAL", time.Hour),
		GroqSTTMaxRequestBytes: getPositiveInt64Env("GROQ_STT_MAX_REQUEST_BYTES", 25_000_000),

		Providers: struct {
			Gemini     APIConfig
			Groq       APIConfig
			OpenRouter APIConfig
		}{
			Gemini: APIConfig{
				APIKey: os.Getenv("GEMINI_API_KEY"),
				BaseURL: getEnv(
					"GEMINI_BASE_URL",
					"https://generativelanguage.googleapis.com/v1beta/openai",
				),
			},

			Groq: APIConfig{
				APIKey: os.Getenv("GROQ_API_KEY"),
				BaseURL: getEnv(
					"GROQ_BASE_URL",
					"https://api.groq.com/openai/v1",
				),
			},

			OpenRouter: APIConfig{
				APIKey: os.Getenv("OPENROUTER_API_KEY"),
				BaseURL: getEnv(
					"OPENROUTER_BASE_URL",
					"https://openrouter.ai/api/v1",
				),
			},
		},
	}
}

func getPositiveInt64Env(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
