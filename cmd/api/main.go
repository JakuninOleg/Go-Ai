package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jakuninoleg/Go-Ai/internal/config"
	"github.com/jakuninoleg/Go-Ai/internal/observability"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
	"github.com/jakuninoleg/Go-Ai/internal/routes"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

func main() {

	cfg := config.Load()
	observer := observability.New(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

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

	aiService := services.NewAIService(
		aiProvider,
	)
	aiService.StartProviderModelCatalogRefresh(
		context.Background(),
		cfg.ModelRefreshInterval,
		func(err error) {
			fmt.Printf("model catalog refresh warning: %v\n", err)
		},
	)

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)

	routes.Register(
		r,
		aiService,
		cfg.SharedSecret,
		observer,
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
