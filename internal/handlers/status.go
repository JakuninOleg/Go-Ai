package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/observability"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

type statusResponse struct {
	observability.Snapshot
	RuntimeGeminiSelection models.RuntimeGeminiSelectionSnapshot `json:"runtime_gemini_selection"`
}

func StatusHandler(service *services.AIService, observer *observability.Observer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := observability.NewMetrics()
		if observer != nil && observer.Metrics != nil {
			metrics = observer.Metrics
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(statusResponse{
			Snapshot:               metrics.Snapshot(),
			RuntimeGeminiSelection: service.RuntimeGeminiSelectionSnapshot(),
		})
	}
}
