package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/observability"
)

func StatusHandler(observer *observability.Observer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := observability.NewMetrics()
		if observer != nil && observer.Metrics != nil {
			metrics = observer.Metrics
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metrics.Snapshot())
	}
}
