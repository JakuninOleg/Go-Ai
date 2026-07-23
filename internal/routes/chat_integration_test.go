package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/observability"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

type httpCaptureProvider struct {
	body       []byte
	statusCode int
	headers    http.Header
	response   []byte
}

func (p *httpCaptureProvider) Chat(_ context.Context, body []byte) (*http.Response, error) {
	p.body = append([]byte(nil), body...)

	return &http.Response{
		StatusCode: p.statusCode,
		Header:     p.headers.Clone(),
		Body:       io.NopCloser(bytes.NewReader(p.response)),
	}, nil
}

func TestChatCompletionsHTTPPassesThroughToolCallingRequestAndResponse(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusAccepted,
		headers: http.Header{
			"Content-Type":       []string{"application/json"},
			"X-Provider-Request": []string{"provider-request-123"},
		},
		response: []byte(`{
			"id":"chatcmpl_123",
			"object":"chat.completion",
			"choices":[{
				"index":0,
				"message":{
					"role":"assistant",
					"content":null,
					"tool_calls":[{
						"id":"call_weather_1",
						"type":"function",
						"function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}
					}]
				},
				"finish_reason":"tool_calls"
			}]
		}`),
	}

	handler := newTestRouter(fakeGemini)
	requestBody := []byte(`{
		"messages":[{"role":"user","content":"What is the weather in Moscow?"}],
		"tools":[{
			"type":"function",
			"function":{
				"name":"get_weather",
				"description":"Get current weather",
				"parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}
			}
		}],
		"tool_choice":"auto",
		"parallel_tool_calls":true
	}`)

	response := postChatCompletion(t, handler, requestBody)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected response content type to be proxied, got %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("X-Provider-Request") != "provider-request-123" {
		t.Fatalf("expected provider header to be proxied, got %q", response.Header().Get("X-Provider-Request"))
	}
	if response.Header().Get(observability.RequestIDHeader) == "" {
		t.Fatal("expected generated request id header")
	}
	if response.Header().Get(observability.DurationHeader) == "" {
		t.Fatal("expected duration header")
	}
	assertRawJSONEqual(t, response.Body.Bytes(), fakeGemini.response)

	var upstream map[string]json.RawMessage
	if err := json.Unmarshal(fakeGemini.body, &upstream); err != nil {
		t.Fatalf("failed to decode captured upstream body: %v", err)
	}

	expectedModel, err := json.Marshal(models.Registry[models.DefaultModelAlias].Name)
	if err != nil {
		t.Fatalf("failed to marshal expected model: %v", err)
	}
	assertRawJSONEqual(t, upstream["model"], expectedModel)
	assertRawJSONEqual(t, upstream["tools"], []byte(`[{"type":"function","function":{"name":"get_weather","description":"Get current weather","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}]`))
	assertRawJSONEqual(t, upstream["tool_choice"], []byte(`"auto"`))
	assertRawJSONEqual(t, upstream["parallel_tool_calls"], []byte(`true`))
}

func TestChatCompletionsPreservesClientRequestID(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers: http.Header{
			observability.RequestIDHeader: []string{"upstream-request-id"},
		},
		response: []byte(`{"ok":true}`),
	}

	handler := newTestRouter(fakeGemini)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{
		"messages":[{"role":"user","content":"hello"}]
	}`)))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(observability.RequestIDHeader, "client-request-123")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Header().Get(observability.RequestIDHeader) != "client-request-123" {
		t.Fatalf("expected client request id to be preserved, got %q", response.Header().Get(observability.RequestIDHeader))
	}
}

func TestChatCompletionsHTTPPassesThroughToolCallHistory(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		response:   []byte(`{"ok":true}`),
	}

	handler := newTestRouter(fakeGemini)
	requestBody := []byte(`{
		"model":"gemini-flash",
		"messages":[
			{"role":"user","content":"What is the weather in Moscow?"},
			{
				"role":"assistant",
				"content":null,
				"tool_calls":[{
					"id":"call_weather_1",
					"type":"function",
					"function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}
				}]
			},
			{"role":"tool","tool_call_id":"call_weather_1","content":"{\"temperature\":\"-5 C\"}"}
		]
	}`)

	response := postChatCompletion(t, handler, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var upstream map[string]json.RawMessage
	if err := json.Unmarshal(fakeGemini.body, &upstream); err != nil {
		t.Fatalf("failed to decode captured upstream body: %v", err)
	}

	expectedMessages := []byte(`[
		{"role":"user","content":"What is the weather in Moscow?"},
		{"role":"assistant","content":null,"tool_calls":[{"id":"call_weather_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}}]},
		{"role":"tool","tool_call_id":"call_weather_1","content":"{\"temperature\":\"-5 C\"}"}
	]`)
	assertRawJSONEqual(t, upstream["messages"], expectedMessages)
}

func TestChatCompletionsHTTPPassesThroughStreamingSSE(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers: http.Header{
			"Content-Type":      []string{"text/event-stream"},
			"Cache-Control":     []string{"no-cache"},
			"Transfer-Encoding": []string{"chunked"},
			"Connection":        []string{"keep-alive"},
		},
		response: []byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n" +
			"data: [DONE]\n\n"),
	}

	handler := newTestRouter(fakeGemini)
	requestBody := []byte(`{
		"model":"gemini-flash",
		"messages":[{"role":"user","content":"Say hello."}],
		"stream":true
	}`)

	response := postChatCompletion(t, handler, requestBody)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected SSE content type to be proxied, got %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected cache-control to be proxied, got %q", response.Header().Get("Cache-Control"))
	}
	if response.Header().Get("Transfer-Encoding") != "" {
		t.Fatalf("expected transfer-encoding hop-by-hop header to be stripped, got %q", response.Header().Get("Transfer-Encoding"))
	}
	if response.Header().Get("Connection") != "" {
		t.Fatalf("expected connection hop-by-hop header to be stripped, got %q", response.Header().Get("Connection"))
	}
	if response.Body.String() != string(fakeGemini.response) {
		t.Fatalf("expected SSE body %q, got %q", fakeGemini.response, response.Body.String())
	}

	var upstream map[string]json.RawMessage
	if err := json.Unmarshal(fakeGemini.body, &upstream); err != nil {
		t.Fatalf("failed to decode captured upstream body: %v", err)
	}
	assertRawJSONEqual(t, upstream["stream"], []byte(`true`))
}

func TestStatusEndpointRequiresAuthAndReturnsMetricsSnapshot(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		response:   []byte(`{"ok":true}`),
	}
	handler, observer, _ := newTestRouterWithObserver(fakeGemini)

	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorizedRequest)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status %d, got %d", http.StatusUnauthorized, unauthorizedResponse.Code)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer test-secret")
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorizedRequest)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, authorizedResponse.Code)
	}
	if authorizedResponse.Header().Get(observability.RequestIDHeader) == "" {
		t.Fatal("expected generated request id header")
	}

	var payload observability.Snapshot
	if err := json.Unmarshal(authorizedResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Status)
	}
	if observer.Metrics.Snapshot().AuthFailuresTotal != 1 {
		t.Fatalf("expected auth failure counter 1, got %d", observer.Metrics.Snapshot().AuthFailuresTotal)
	}
}

func TestObservabilityMetricsAndSafeChatLog(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		response:   []byte(`{"ok":true}`),
	}
	handler, observer, logs := newTestRouterWithObserver(fakeGemini)
	requestBody := []byte(`{
		"model":"gemini-flash",
		"messages":[{"role":"user","content":"secret prompt text that must not be logged"}],
		"stream":true
	}`)

	response := postChatCompletion(t, handler, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	snapshot := observer.Metrics.Snapshot()
	if snapshot.RequestsTotal != 1 {
		t.Fatalf("expected requests_total 1, got %d", snapshot.RequestsTotal)
	}
	if snapshot.ChatRequestsTotal != 1 {
		t.Fatalf("expected chat_requests_total 1, got %d", snapshot.ChatRequestsTotal)
	}
	if snapshot.SuccessTotal != 1 {
		t.Fatalf("expected success_total 1, got %d", snapshot.SuccessTotal)
	}
	if snapshot.StreamingRequestsTotal != 1 {
		t.Fatalf("expected streaming_requests_total 1, got %d", snapshot.StreamingRequestsTotal)
	}
	if snapshot.ProviderRequests[models.Registry[models.DefaultModelAlias].Provider] != 1 {
		t.Fatalf("expected provider request counter, got %#v", snapshot.ProviderRequests)
	}
	if snapshot.StatusCodes["200"] != 1 {
		t.Fatalf("expected status code counter, got %#v", snapshot.StatusCodes)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, `"request_id"`) {
		t.Fatalf("expected log to include request_id, got %s", logOutput)
	}
	if strings.Contains(logOutput, "secret prompt text that must not be logged") {
		t.Fatalf("expected log not to include prompt text, got %s", logOutput)
	}
}

func TestModelsEndpointRequiresAuthAndReturnsSafeStatus(t *testing.T) {
	fakeGemini := &httpCaptureProvider{
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		response:   []byte(`{"ok":true}`),
	}
	handler := newTestRouter(fakeGemini)

	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorizedRequest)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status %d, got %d", http.StatusUnauthorized, unauthorizedResponse.Code)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer test-secret")
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorizedRequest)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, authorizedResponse.Code)
	}

	var payload struct {
		DefaultAlias string `json:"default_alias"`
		Aliases      map[string][]struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"aliases"`
	}
	if err := json.Unmarshal(authorizedResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode models response: %v", err)
	}
	if payload.DefaultAlias != models.DefaultModelAlias {
		t.Fatalf("expected default alias %q, got %q", models.DefaultModelAlias, payload.DefaultAlias)
	}
	if len(payload.Aliases[models.DefaultModelAlias]) < 2 {
		t.Fatalf("expected default alias fallback candidates, got %#v", payload.Aliases[models.DefaultModelAlias])
	}
}

func newTestRouter(gemini providers.Provider) http.Handler {
	handler, _, _ := newTestRouterWithObserver(gemini)
	return handler
}

func newTestRouterWithObserver(gemini providers.Provider) (http.Handler, *observability.Observer, *bytes.Buffer) {
	router := chi.NewRouter()
	service := services.NewAIService(providers.NewProviderRouter(gemini, &httpCaptureProvider{}))
	logs := &bytes.Buffer{}
	observer := observability.New(slog.New(slog.NewJSONHandler(logs, nil)))
	Register(router, service, "test-secret", observer)

	return router, observer, logs
}

func postChatCompletion(t *testing.T, handler http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func assertRawJSONEqual(t *testing.T, actual []byte, expected []byte) {
	t.Helper()

	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("failed to decode actual JSON %s: %v", actual, err)
	}

	var expectedValue any
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("failed to decode expected JSON %s: %v", expected, err)
	}

	actualJSON, err := json.Marshal(actualValue)
	if err != nil {
		t.Fatalf("failed to marshal actual JSON: %v", err)
	}
	expectedJSON, err := json.Marshal(expectedValue)
	if err != nil {
		t.Fatalf("failed to marshal expected JSON: %v", err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatalf("expected JSON %s, got %s", expected, actual)
	}
}
