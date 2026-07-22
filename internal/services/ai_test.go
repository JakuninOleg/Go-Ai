package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

type captureProvider struct {
	body []byte
}

func (p *captureProvider) Chat(_ context.Context, body []byte) (*http.Response, error) {
	p.body = body

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
	}, nil
}

func TestChatUsesDefaultModelWhenMissing(t *testing.T) {
	gemini := &captureProvider{}
	service := NewAIService(providers.NewProviderRouter(gemini, &captureProvider{}))

	_, err := service.Chat(context.Background(), []byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var request map[string]any
	if err := json.Unmarshal(gemini.body, &request); err != nil {
		t.Fatalf("failed to decode captured body: %v", err)
	}

	expectedModel := models.Registry[models.DefaultModelAlias].Name
	if request["model"] != expectedModel {
		t.Fatalf("expected model %q, got %q", expectedModel, request["model"])
	}
}

func TestChatReturnsUnknownModelError(t *testing.T) {
	gemini := &captureProvider{}
	service := NewAIService(providers.NewProviderRouter(gemini, &captureProvider{}))

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
	service := NewAIService(providers.NewProviderRouter(gemini, &captureProvider{}))

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
