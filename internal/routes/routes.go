package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/jakuninoleg/Go-Ai/internal/handlers"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

func Register(
	r chi.Router,
	aiProvider providers.Provider,
) {

	r.Get("/health", handlers.Health)

	r.Post(
		"/v1/chat/completions",
		handlers.ChatHandler(aiProvider),
	)

}
