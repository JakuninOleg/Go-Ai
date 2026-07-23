package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

func TestOpenRouterListModelsParsesModelsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("expected /models path, got %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"},{"id":"model-b"},{"id":""}]}`))
	}))
	defer server.Close()

	provider := NewOpenRouterProvider(config.APIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model-a" || models[1].ID != "model-b" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestGeminiListModelsParsesOpenAICompatibleModelsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/openai/models" {
			t.Fatalf("expected Gemini OpenAI-compatible models path, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gemini-3.5-flash"}]}`))
	}))
	defer server.Close()

	provider := NewGeminiProvider(config.APIConfig{
		APIKey:  "test-key",
		BaseURL: server.URL + "/v1beta/openai",
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(models) != 1 || models[0].ID != "gemini-3.5-flash" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestProviderRouterRefreshModelCatalogCachesSuccessAndErrors(t *testing.T) {
	gemini := &modelListerProvider{models: []ModelInfo{{ID: "gemini-3.5-flash"}}}
	openRouter := &modelListerProvider{err: errTestDiscovery}
	router := NewProviderRouter(gemini, openRouter)

	if err := router.RefreshModelCatalog(context.Background()); err == nil {
		t.Fatal("expected refresh to return the discovery error")
	}

	snapshot := router.ModelCatalogSnapshot()
	if snapshot.LastSuccessfulRefresh.IsZero() {
		t.Fatal("expected last successful refresh time to be set")
	}
	if len(snapshot.Providers["gemini"].Models) != 1 {
		t.Fatalf("expected cached gemini model, got %#v", snapshot.Providers["gemini"].Models)
	}
	if snapshot.Providers["openrouter"].LastError == "" {
		t.Fatal("expected openrouter refresh error to be stored")
	}
	if router.IsKnownUnavailable("gemini", "missing-model") != true {
		t.Fatal("expected missing model to be known unavailable after catalog refresh")
	}
	if router.IsKnownUnavailable("gemini", "gemini-3.5-flash") != false {
		t.Fatal("expected discovered model to be available")
	}
}

type modelListerProvider struct {
	models []ModelInfo
	err    error
}

func (p *modelListerProvider) Chat(context.Context, []byte) (*http.Response, error) {
	return nil, nil
}

func (p *modelListerProvider) ListModels(context.Context) ([]ModelInfo, error) {
	if p.err != nil {
		return nil, p.err
	}

	return p.models, nil
}

type testDiscoveryError struct{}

func (testDiscoveryError) Error() string {
	return "test discovery error"
}

var errTestDiscovery = testDiscoveryError{}
