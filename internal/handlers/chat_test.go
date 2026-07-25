package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

func TestWriteServiceErrorClassifiesUpstreamFailureWithoutLeakingDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := providers.ClassifyUpstreamError("gemini", errors.New("dial tcp private-provider: connection refused"))

	status, errorType := writeServiceError(recorder, err)
	if status != http.StatusBadGateway || errorType != "provider_transport" {
		t.Fatalf("unexpected status/error type: %d %q", status, errorType)
	}

	var response errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error.Code != "provider_transport" || response.Error.Message != "AI provider request failed" {
		t.Fatalf("unexpected error response: %#v", response.Error)
	}
	if string(recorder.Body.Bytes()) == "" || containsPrivateDetail(string(recorder.Body.Bytes())) {
		t.Fatalf("response leaked provider error details: %s", recorder.Body.Bytes())
	}
}

func containsPrivateDetail(value string) bool {
	return strings.Contains(value, "dial tcp private-provider: connection refused")
}
