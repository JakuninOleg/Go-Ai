package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/JakuninOleg/go-platform/internal/routes"
)


func main() {

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	routes.Register(r)


	port := ":8080"

	fmt.Printf("Server starting on http://localhost%s\n", port)

	http.ListenAndServe(port, r)
}