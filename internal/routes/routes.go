package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/jakuninoleg/Go-Ai/internal/handlers"
)


func Register(r chi.Router) {

	r.Get("/health", handlers.Health)

}