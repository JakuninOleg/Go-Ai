package handlers

import (
	"io"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

func ChatHandler(
	provider providers.Provider,
) http.HandlerFunc {

	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		body, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(
				w,
				"failed to read request body",
				http.StatusBadRequest,
			)
			return
		}

		resp, err := provider.Chat(
			r.Context(),
			body,
		)

		if err != nil {
			http.Error(
				w,
				"AI provider error",
				http.StatusBadGateway,
			)
			return
		}

		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)

		io.Copy(
			w,
			resp.Body,
		)
	}
}
