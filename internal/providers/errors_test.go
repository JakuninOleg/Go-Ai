package providers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

func TestClassifyUpstreamErrorUsesSafeCategories(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		category string
	}{
		{name: "context canceled", err: context.Canceled, category: "context_canceled"},
		{name: "deadline exceeded", err: context.DeadlineExceeded, category: "timeout"},
		{name: "DNS", err: &net.DNSError{Err: "not found", Name: "provider.example"}, category: "dns"},
		{name: "transport", err: errors.New("connection refused"), category: "transport"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ClassifyUpstreamError("gemini", testCase.err)

			var upstreamErr UpstreamError
			if !errors.As(err, &upstreamErr) {
				t.Fatalf("expected UpstreamError, got %T", err)
			}
			if upstreamErr.Provider != "gemini" || upstreamErr.Category != testCase.category {
				t.Fatalf("unexpected upstream error: %#v", upstreamErr)
			}
			if errors.Unwrap(upstreamErr) != testCase.err {
				t.Fatal("expected original error to be preserved for programmatic handling")
			}
		})
	}
}

func TestGeminiChatClassifiesHTTPClientFailureWithoutLeakingRequestDetails(t *testing.T) {
	provider := NewGeminiProvider(config.APIConfig{
		APIKey:  "test-key",
		BaseURL: "https://provider.example/v1beta/openai",
	})
	provider.client = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Post", URL: "https://provider.example/private", Err: context.DeadlineExceeded}
	})}

	_, err := provider.Chat(context.Background(), []byte(`{"model":"gemini-test-model"}`))
	if err == nil {
		t.Fatal("expected Chat to return an error")
	}

	var upstreamErr UpstreamError
	if !errors.As(err, &upstreamErr) {
		t.Fatalf("expected UpstreamError, got %T", err)
	}
	if upstreamErr.Provider != "gemini" || upstreamErr.Category != "timeout" {
		t.Fatalf("unexpected upstream error: %#v", upstreamErr)
	}
	if got := err.Error(); got != "upstream timeout error for provider: gemini" {
		t.Fatalf("unexpected public error string: %q", got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
