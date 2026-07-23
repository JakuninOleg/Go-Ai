package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/observability"
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
	observers ...*observability.Observer,
) http.HandlerFunc {
	observer := firstObserver(observers...)

	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		start := time.Now()

		body, err := io.ReadAll(r.Body)

		if err != nil {
			logChat(observer, r, chatLogFields{
				status:      http.StatusBadRequest,
				duration:    time.Since(start),
				errorType:   "invalid_request_body",
				modelAlias:  models.DefaultModelAlias,
				stream:      false,
				fallbackSet: true,
			})
			writeJSONError(
				w,
				"failed to read request body",
				"invalid_request_error",
				"invalid_request_body",
				http.StatusBadRequest,
			)
			return
		}
		modelAlias, stream := chatRequestMetadata(body)
		recordChatRequest(observer, stream)

		resp, err := service.Chat(
			r.Context(),
			body,
		)

		if err != nil {
			status, errorType := writeServiceError(w, err)
			logChat(observer, r, chatLogFields{
				status:      status,
				duration:    time.Since(start),
				modelAlias:  modelAlias,
				stream:      stream,
				errorType:   errorType,
				fallbackSet: true,
			})
			return
		}

		defer resp.Body.Close()

		copyResponseHeaders(w.Header(), resp.Header)
		recordProviderRequest(observer, resp.Header)

		w.WriteHeader(resp.StatusCode)

		copyErr := copyResponseBody(
			w,
			resp.Body,
		)

		logChat(observer, r, chatLogFields{
			status:        resp.StatusCode,
			duration:      time.Since(start),
			modelAlias:    responseHeaderOr(resp.Header, "X-Go-Ai-Model-Alias", modelAlias),
			provider:      resp.Header.Get("X-Go-Ai-Provider"),
			upstreamModel: resp.Header.Get("X-Go-Ai-Upstream-Model"),
			fallbackUsed:  resp.Header.Get("X-Go-Ai-Fallback-Used") == "true",
			fallbackSet:   resp.Header.Get("X-Go-Ai-Fallback-Used") != "",
			stream:        stream,
			errorType:     copyErrorType(copyErr),
		})
	}
}

type chatLogFields struct {
	status        int
	duration      time.Duration
	modelAlias    string
	provider      string
	upstreamModel string
	fallbackUsed  bool
	fallbackSet   bool
	stream        bool
	errorType     string
}

func chatRequestMetadata(body []byte) (string, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return models.DefaultModelAlias, false
	}

	modelAlias := models.DefaultModelAlias
	if rawModel, ok := fields["model"]; ok {
		var requestedModel string
		if err := json.Unmarshal(rawModel, &requestedModel); err == nil {
			modelAlias = requestedModel
		}
	}

	var stream bool
	if rawStream, ok := fields["stream"]; ok {
		_ = json.Unmarshal(rawStream, &stream)
	}

	return modelAlias, stream
}

func recordChatRequest(observer *observability.Observer, stream bool) {
	if observer == nil || observer.Metrics == nil {
		return
	}
	observer.Metrics.RecordChatRequest(stream)
}

func recordProviderRequest(observer *observability.Observer, header http.Header) {
	if observer == nil || observer.Metrics == nil {
		return
	}
	observer.Metrics.RecordProviderRequest(
		header.Get("X-Go-Ai-Provider"),
		header.Get("X-Go-Ai-Fallback-Used") == "true",
	)
}

func logChat(observer *observability.Observer, r *http.Request, fields chatLogFields) {
	if observer == nil || observer.Logger == nil {
		return
	}

	attrs := []slog.Attr{
		slog.String("event", "chat_request"),
		slog.String("request_id", observability.RequestIDFromContext(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", fields.status),
		slog.Int64("duration_ms", fields.duration.Milliseconds()),
		slog.String("model_alias", fields.modelAlias),
		slog.Bool("stream", fields.stream),
	}

	if fields.provider != "" {
		attrs = append(attrs, slog.String("provider", fields.provider))
	}
	if fields.upstreamModel != "" {
		attrs = append(attrs, slog.String("upstream_model", fields.upstreamModel))
	}
	if fields.fallbackSet {
		attrs = append(attrs, slog.Bool("fallback_used", fields.fallbackUsed))
	}
	if fields.errorType != "" {
		attrs = append(attrs, slog.String("error_type", fields.errorType))
		observer.Logger.LogAttrs(r.Context(), slog.LevelError, "chat request completed", attrs...)
		return
	}

	observer.Logger.LogAttrs(r.Context(), slog.LevelInfo, "chat request completed", attrs...)
}

func responseHeaderOr(header http.Header, key string, fallback string) string {
	value := header.Get(key)
	if value == "" {
		return fallback
	}
	return value
}

func copyErrorType(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errResponseWrite) {
		return "response_write_error"
	}
	return "upstream_read_error"
}

func copyResponseHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		if strings.EqualFold(key, observability.RequestIDHeader) {
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

var errResponseWrite = errors.New("response write error")

func copyResponseBody(w http.ResponseWriter, body io.Reader) error {
	flusher, canFlush := w.(http.Flusher)
	buffer := make([]byte, 32*1024)

	for {
		n, readErr := body.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return errors.Join(errResponseWrite, writeErr)
			}
			if canFlush {
				flusher.Flush()
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

func writeServiceError(w http.ResponseWriter, err error) (int, string) {
	var unknownModelErr models.UnknownModelError
	if errors.As(err, &unknownModelErr) {
		writeJSONError(
			w,
			"unknown model: "+unknownModelErr.Alias,
			"invalid_request_error",
			"unknown_model",
			http.StatusBadRequest,
		)
		return http.StatusBadRequest, "unknown_model"
	}

	if errors.Is(err, services.ErrInvalidJSON) {
		writeJSONError(
			w,
			"invalid JSON request body",
			"invalid_request_error",
			"invalid_json",
			http.StatusBadRequest,
		)
		return http.StatusBadRequest, "invalid_json"
	}
	if errors.Is(err, services.ErrInvalidRequestObject) {
		writeJSONError(
			w,
			"request body must be a JSON object",
			"invalid_request_error",
			"invalid_request_body",
			http.StatusBadRequest,
		)
		return http.StatusBadRequest, "invalid_request_body"
	}
	if errors.Is(err, services.ErrModelMustBeString) {
		writeJSONError(
			w,
			"model must be a string",
			"invalid_request_error",
			"invalid_model",
			http.StatusBadRequest,
		)
		return http.StatusBadRequest, "invalid_model"
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
		return http.StatusBadGateway, "provider_not_configured"
	}

	writeJSONError(
		w,
		"AI provider error",
		"server_error",
		"provider_error",
		http.StatusBadGateway,
	)
	return http.StatusBadGateway, "provider_error"
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
