package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type APIConfig struct {
	APIKey  string
	BaseURL string
}

type Config struct {
	Port                 string
	SharedSecret         string
	ModelRefreshInterval time.Duration

	Providers struct {
		Gemini     APIConfig
		OpenRouter APIConfig
	}
}

func Load() Config {
	godotenv.Load()

	return Config{
		Port:                 getEnv("PORT", "8080"),
		SharedSecret:         os.Getenv("GO_AI_SHARED_SECRET"),
		ModelRefreshInterval: getDurationEnv("MODEL_REFRESH_INTERVAL", time.Hour),

		Providers: struct {
			Gemini     APIConfig
			OpenRouter APIConfig
		}{
			Gemini: APIConfig{
				APIKey: os.Getenv("GEMINI_API_KEY"),
				BaseURL: getEnv(
					"GEMINI_BASE_URL",
					"https://generativelanguage.googleapis.com/v1beta/openai",
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
