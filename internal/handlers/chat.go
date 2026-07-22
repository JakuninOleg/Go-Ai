package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

type errorResponse struct {
	Error errorDetails `json:"error"`
}

type errorDetails struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

func ChatHandler(
	service *services.AIService,
) http.HandlerFunc {

	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		body, err := io.ReadAll(r.Body)

		if err != nil {
			writeJSONError(
				w,
				"failed to read request body",
				"invalid_request_error",
				"invalid_request_body",
				http.StatusBadRequest,
			)
			return
		}

		resp, err := service.Chat(
			r.Context(),
			body,
		)

		if err != nil {
			writeServiceError(w, err)
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

func writeServiceError(w http.ResponseWriter, err error) {
	var unknownModelErr models.UnknownModelError
	if errors.As(err, &unknownModelErr) {
		writeJSONError(
			w,
			"unknown model: "+unknownModelErr.Alias,
			"invalid_request_error",
			"unknown_model",
			http.StatusBadRequest,
		)
		return
	}

	if errors.Is(err, services.ErrInvalidJSON) {
		writeJSONError(
			w,
			"invalid JSON request body",
			"invalid_request_error",
			"invalid_json",
			http.StatusBadRequest,
		)
		return
	}
	if errors.Is(err, services.ErrInvalidRequestObject) {
		writeJSONError(
			w,
			"request body must be a JSON object",
			"invalid_request_error",
			"invalid_request_body",
			http.StatusBadRequest,
		)
		return
	}
	if errors.Is(err, services.ErrModelMustBeString) {
		writeJSONError(
			w,
			"model must be a string",
			"invalid_request_error",
			"invalid_model",
			http.StatusBadRequest,
		)
		return
	}

	var missingAPIKeyErr providers.MissingAPIKeyError
	if errors.As(err, &missingAPIKeyErr) {
		writeJSONError(
			w,
			"provider is not configured",
			"server_error",
			"provider_not_configured",
			http.StatusBadGateway,
		)
		return
	}

	writeJSONError(
		w,
		"AI provider error",
		"server_error",
		"provider_error",
		http.StatusBadGateway,
	)
}

func writeJSONError(
	w http.ResponseWriter,
	message string,
	errorType string,
	code string,
	status int,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: errorDetails{
			Message: message,
			Type:    errorType,
			Code:    code,
		},
	})
}
