package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

type captureProvider struct {
	body []byte
}

func (p *captureProvider) ListModels(context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "gemini-3.10-flash"}}, nil
}

func (p *captureProvider) Chat(_ context.Context, body []byte) (*http.Response, error) {
	p.body = body

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
	}, nil
}

type sequenceProvider struct {
	calls     int
	bodies    [][]byte
	responses []providerResult
}

func (p *sequenceProvider) ListModels(context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{{ID: "gemini-3.10-flash"}}, nil
}

type providerResult struct {
	status int
	body   string
	err    error
}

type catalogProvider struct {
	*sequenceProvider
	models []providers.ModelInfo
}

func (p *catalogProvider) ListModels(context.Context) ([]providers.ModelInfo, error) {
	return p.models, nil
}

func (p *sequenceProvider) Chat(_ context.Context, body []byte) (*http.Response, error) {
	p.calls++
	p.bodies = append(p.bodies, append([]byte(nil), body...))

	result := p.responses[p.calls-1]
	if result.err != nil {
		return nil, result.err
	}

	return &http.Response{
		StatusCode: result.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader([]byte(result.body))),
	}, nil
}

func newSelectedAIService(gemini, openRouter providers.Provider) *AIService {
	router := providers.NewProviderRouter(gemini, openRouter)
	if err := router.RefreshModelCatalog(context.Background()); err != nil {
		panic(err)
	}
	return NewAIService(router)
}

func TestChatUsesDefaultModelWhenMissing(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	_, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]any
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	expectedModel := "gemini-3.10-flash"
	if request["model"] != expectedModel {
		t.Fatalf("expected model %q, got %q", expectedModel, request["model"])
	}
}

func TestChatPreservesToolCallingFieldsWithDefaultModel(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	body := []byte(`{
		"messages":[{"role":"user","content":"What is the weather?"}],
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

	_, err := service.Chat(context.Background(), body)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]json.RawMessage
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	expectedModel, err := json.Marshal("gemini-3.10-flash")
	if err != nil {
		t.Fatalf("failed to marshal expected model: %v", err)
	}
	if string(request["model"]) != string(expectedModel) {
		t.Fatalf("expected model %s, got %s", expectedModel, request["model"])
	}
	assertRawJSONEqual(t, request["tools"], []byte(`[{"type":"function","function":{"name":"get_weather","description":"Get current weather","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}]`))
	assertRawJSONEqual(t, request["tool_choice"], []byte(`"auto"`))
	assertRawJSONEqual(t, request["parallel_tool_calls"], []byte(`true`))
}

func TestChatPreservesStreamingFlagAndRewritesModel(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	body := []byte(`{
		"model":"default",
		"messages":[{"role":"user","content":"Stream this response."}],
		"stream":true
	}`)

	_, err := service.Chat(context.Background(), body)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]json.RawMessage
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	expectedModel, err := json.Marshal("gemini-3.10-flash")
	if err != nil {
		t.Fatalf("failed to marshal expected model: %v", err)
	}
	assertRawJSONEqual(t, request["model"], expectedModel)
	assertRawJSONEqual(t, request["stream"], []byte(`true`))
}

func TestChatPreservesAssistantToolCalls(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	body := []byte(`{
		"model":"default",
		"messages":[{
			"role":"assistant",
			"content":null,
			"tool_calls":[{
				"id":"call_123",
				"type":"function",
				"function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}
			}]
		}]
	}`)

	_, err := service.Chat(context.Background(), body)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]json.RawMessage
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	assertRawJSONEqual(t, request["messages"], []byte(`[{"role":"assistant","content":null,"tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}}]}]`))
}

func TestChatPreservesToolRoleMessage(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	body := []byte(`{
		"model":"default",
		"messages":[{
			"role":"tool",
			"tool_call_id":"call_123",
			"content":"{\"temperature\":\"-5 C\"}"
		}]
	}`)

	_, err := service.Chat(context.Background(), body)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]json.RawMessage
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	assertRawJSONEqual(t, request["messages"], []byte(`[{"role":"tool","tool_call_id":"call_123","content":"{\"temperature\":\"-5 C\"}"}]`))
}

func TestChatUsesGeminiForDefault(t *testing.T) {
	gemini := &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"gemini"}`}}}
	openRouter := &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"openrouter"}`}}}
	service := newSelectedAIService(gemini, openRouter)

	resp, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	defer resp.Body.Close()

	if gemini.calls != 1 {
		t.Fatalf("expected Gemini to be called once, got %d calls", gemini.calls)
	}
	if openRouter.calls != 0 {
		t.Fatalf("expected OpenRouter not to be called, got %d", openRouter.calls)
	}
	if resp.Header.Get("X-Go-Ai-Fallback-Used") != "false" {
		t.Fatalf("expected fallback header false, got %q", resp.Header.Get("X-Go-Ai-Fallback-Used"))
	}
}

func TestChatUsesDefaultCandidatePresentInCatalog(t *testing.T) {
	gemini := &catalogProvider{
		sequenceProvider: &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"gemini"}`}}},
		models:           []providers.ModelInfo{{ID: "gemini-3.7-flash"}},
	}
	openRouter := &catalogProvider{
		sequenceProvider: &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"openrouter"}`}}},
		models:           []providers.ModelInfo{{ID: "openrouter/free"}},
	}
	router := providers.NewProviderRouter(gemini, openRouter)
	if err := router.RefreshModelCatalog(context.Background()); err != nil {
		t.Fatalf("RefreshModelCatalog returned error: %v", err)
	}
	service := NewAIService(router)

	resp, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	defer resp.Body.Close()

	if gemini.calls != 1 || resp.Header.Get("X-Go-Ai-Provider") != models.ProviderGemini {
		t.Fatalf("expected Gemini default candidate, calls=%d headers=%#v", gemini.calls, resp.Header)
	}
	if openRouter.calls != 0 {
		t.Fatalf("expected OpenRouter not to be called, got %d", openRouter.calls)
	}
}

func TestChatUsesOpenRouterFallbackWhenGeminiMissingFromCatalog(t *testing.T) {
	gemini := &catalogProvider{
		sequenceProvider: &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"gemini"}`}}},
		models:           []providers.ModelInfo{{ID: "another-gemini-model"}},
	}
	openRouter := &catalogProvider{
		sequenceProvider: &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"openrouter"}`}}},
		models:           []providers.ModelInfo{{ID: "openrouter/free"}},
	}
	router := providers.NewProviderRouter(gemini, openRouter)
	if err := router.RefreshModelCatalog(context.Background()); err != nil {
		t.Fatalf("RefreshModelCatalog returned error: %v", err)
	}
	service := NewAIService(router)

	resp, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	defer resp.Body.Close()

	if gemini.calls != 0 {
		t.Fatalf("expected Gemini to be skipped by catalog, got %d calls", gemini.calls)
	}
	if openRouter.calls != 1 || resp.Header.Get("X-Go-Ai-Provider") != models.ProviderOpenRouter {
		t.Fatalf("expected OpenRouter fallback candidate, calls=%d headers=%#v", openRouter.calls, resp.Header)
	}
}

func TestChatReturnsUnavailableErrorForGeminiFlashWithoutCatalogCandidate(t *testing.T) {
	gemini := &catalogProvider{
		sequenceProvider: &sequenceProvider{},
		models:           []providers.ModelInfo{{ID: "gemini-3.7-pro"}},
	}
	openRouter := &catalogProvider{sequenceProvider: &sequenceProvider{}}
	router := providers.NewProviderRouter(gemini, openRouter)
	if err := router.RefreshModelCatalog(context.Background()); err != nil {
		t.Fatalf("RefreshModelCatalog returned error: %v", err)
	}

	_, err := NewAIService(router).Chat(context.Background(), []byte(`{"model":"gemini-flash","messages":[]}`))
	var unavailableErr models.ModelUnavailableError
	if !errors.As(err, &unavailableErr) {
		t.Fatalf("expected ModelUnavailableError, got %v", err)
	}
}

func TestChatReturnsDefaultCandidateRetryableResponse(t *testing.T) {
	gemini := &sequenceProvider{responses: []providerResult{{status: http.StatusGatewayTimeout, body: `{"error":"timeout"}`}}}
	openRouter := &sequenceProvider{responses: []providerResult{{status: http.StatusGatewayTimeout, body: `{"error":"timeout"}`}}}
	service := newSelectedAIService(gemini, openRouter)

	resp, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected final upstream status %d, got %d", http.StatusGatewayTimeout, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read final response body: %v", err)
	}
	if string(body) != `{"error":"timeout"}` {
		t.Fatalf("expected final response body, got %s", body)
	}
}

func TestChatDefaultPreservesToolCallingAndStreamingFields(t *testing.T) {
	gemini := &sequenceProvider{responses: []providerResult{{status: http.StatusOK, body: `{"provider":"gemini"}`}}}
	service := newSelectedAIService(gemini, &sequenceProvider{})

	body := []byte(`{
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}}]},
			{"role":"tool","tool_call_id":"call_123","content":"{\"temperature\":\"-5 C\"}"}
		],
		"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object"}}}],
		"tool_choice":"auto",
		"stream":true
	}`)

	resp, err := service.Chat(context.Background(), body)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	defer resp.Body.Close()

	var request map[string]json.RawMessage
	if err := json.Unmarshal(gemini.bodies[0], &request); err != nil {
		t.Fatalf("failed to decode default body: %v", err)
	}

	assertRawJSONEqual(t, request["stream"], []byte(`true`))
	assertRawJSONEqual(t, request["tool_choice"], []byte(`"auto"`))
	assertRawJSONEqual(t, request["tools"], []byte(`[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object"}}}]`))
	assertRawJSONEqual(t, request["messages"], []byte(`[
		{"role":"assistant","content":null,"tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Moscow\"}"}}]},
		{"role":"tool","tool_call_id":"call_123","content":"{\"temperature\":\"-5 C\"}"}
	]`))
}

func TestChatReturnsUnknownModelError(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	_, err := service.Chat(context.Background(), []byte(`{"model":"does-not-exist","messages":[]}`))
	if err == nil {
		t.Fatal("expected error")
	}

	if _, ok := err.(models.UnknownModelError); !ok {
		t.Fatalf("expected UnknownModelError, got %T", err)
	}
	if gemini.body != nil {
		t.Fatal("provider should not be called for unknown model")
	}
}

func TestChatReturnsUnknownModelErrorWhenModelIsEmpty(t *testing.T) {
	gemini := &captureProvider{}
	service := newSelectedAIService(gemini, &captureProvider{})

	_, err := service.Chat(context.Background(), []byte(`{"model":"","messages":[]}`))
	if err == nil {
		t.Fatal("expected error")
	}

	unknownModelErr, ok := err.(models.UnknownModelError)
	if !ok {
		t.Fatalf("expected UnknownModelError, got %T", err)
	}
	if unknownModelErr.Alias != "" {
		t.Fatalf("expected empty alias, got %q", unknownModelErr.Alias)
	}
}

func assertRawJSONEqual(t *testing.T, actual json.RawMessage, expected []byte) {
	t.Helper()

	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("failed to decode actual JSON %s: %v", actual, err)
	}

	var expectedValue any
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("failed to decode expected JSON %s: %v", expected, err)
	}

	if !jsonValuesEqual(actualValue, expectedValue) {
		t.Fatalf("expected JSON %s, got %s", expected, actual)
	}
}

func jsonValuesEqual(a any, b any) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}

	return bytes.Equal(aJSON, bJSON)
}
