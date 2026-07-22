package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

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

		copyResponseHeaders(w.Header(), resp.Header)

		w.WriteHeader(resp.StatusCode)

		copyResponseBody(
			w,
			resp.Body,
		)
	}
}

func copyResponseHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailer",
		"transfer-encoding",
		"upgrade":
		return true
	default:
		return false
	}
}

func copyResponseBody(w http.ResponseWriter, body io.Reader) {
	flusher, canFlush := w.(http.Flusher)
	buffer := make([]byte, 32*1024)

	for {
		n, readErr := body.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}

		if readErr != nil {
			return
		}
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
