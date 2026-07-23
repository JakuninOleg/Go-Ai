package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/jakuninoleg/Go-Ai/internal/handlers"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

func Register(
	r chi.Router,
	aiService *services.AIService,
	sharedSecret string,
) {

	r.Get(
		"/health",
		handlers.Health,
	)

	r.Group(func(r chi.Router) {
		r.Use(handlers.BearerAuth(sharedSecret))

		r.Post(
			"/v1/chat/completions",
			handlers.ChatHandler(aiService),
		)

		r.Get(
			"/v1/models",
			handlers.ModelsHandler(aiService),
		)
	})
}
