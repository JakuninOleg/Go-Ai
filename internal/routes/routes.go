package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/JakuninOleg/go-platform/internal/handlers"
)


func Register(r chi.Router) {

	r.Get("/health", handlers.Health)

}