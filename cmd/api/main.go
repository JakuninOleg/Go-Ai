package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jakuninoleg/Go-Ai/internal/routes"
	"github.com/jakuninoleg/Go-Ai/internal/config"
)


func main() {

	cfg := config.Load()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	routes.Register(r)

	port := ":" + cfg.Port

	fmt.Printf("Server starting on http://localhost%s\n", port)

	if err := http.ListenAndServe(port, r); err != nil {
    fmt.Printf("Server failed: %v\n", err)
}
}