package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jakuninoleg/Go-Ai/internal/config"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
	"github.com/jakuninoleg/Go-Ai/internal/routes"
)

func main() {

	cfg := config.Load()

	geminiProvider := providers.NewGeminiProvider(
		cfg.Providers.Gemini,
	)

	openRouterProvider := providers.NewOpenRouterProvider(
		cfg.Providers.OpenRouter,
	)

	aiProvider := providers.NewProviderRouter(
		geminiProvider,
		openRouterProvider,
	)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	routes.Register(
		r,
		aiProvider,
	)

	port := ":" + cfg.Port

	fmt.Printf(
		"Server running on :%s\n",
		cfg.Port,
	)

	if err := http.ListenAndServe(port, r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
