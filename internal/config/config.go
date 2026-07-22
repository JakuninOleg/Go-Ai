package config

import (
	"os"

	"github.com/joho/godotenv"
)

type APIConfig struct {
	APIKey  string
	BaseURL string
}

type Config struct {
	Port         string
	SharedSecret string

	Providers struct {
		Gemini     APIConfig
		OpenRouter APIConfig
	}
}

func Load() Config {
	godotenv.Load()

	return Config{
		Port:         getEnv("PORT", "8080"),
		SharedSecret: os.Getenv("GO_AI_SHARED_SECRET"),

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

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
