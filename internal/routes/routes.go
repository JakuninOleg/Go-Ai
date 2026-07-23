package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/jakuninoleg/Go-Ai/internal/handlers"
	"github.com/jakuninoleg/Go-Ai/internal/observability"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

func Register(
	r chi.Router,
	aiService *services.AIService,
	sharedSecret string,
	observers ...*observability.Observer,
) {
	observer := firstObserver(observers...)
	if observer == nil {
		observer = observability.New(nil)
	}

	r.Use(observability.RequestID)
	r.Use(observability.HTTPMetrics(observer.Metrics))

	r.Get(
		"/health",
		handlers.Health,
	)

	r.Group(func(r chi.Router) {
		r.Use(handlers.BearerAuth(sharedSecret, observer))

		r.Post(
			"/v1/chat/completions",
			handlers.ChatHandler(aiService, observer),
		)

		r.Get(
			"/v1/status",
			handlers.StatusHandler(observer),
		)

		r.Get(
			"/v1/models",
			handlers.ModelsHandler(aiService),
		)
	})
}

func firstObserver(observers ...*observability.Observer) *observability.Observer {
	if len(observers) == 0 {
		return nil
	}
	return observers[0]
}
