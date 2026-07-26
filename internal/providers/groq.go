package providers

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

type AudioProvider interface {
	Transcribe(ctx context.Context, contentType string, contentLength int64, body io.ReadCloser) (*http.Response, error)
	Speech(ctx context.Context, contentType string, contentLength int64, body io.ReadCloser) (*http.Response, error)
}

type GroqProvider struct {
	cfg    config.APIConfig
	client *http.Client
}

func NewGroqProvider(cfg config.APIConfig) *GroqProvider {
	return &GroqProvider{
		cfg:    cfg,
		client: &http.Client{},
	}
}

func (p *GroqProvider) Transcribe(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return p.proxyAudioRequest(ctx, "/audio/transcriptions", contentType, contentLength, body)
}

func (p *GroqProvider) Speech(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return p.proxyAudioRequest(ctx, "/audio/speech", contentType, contentLength, body)
}

func (p *GroqProvider) proxyAudioRequest(
	ctx context.Context,
	path string,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return nil, MissingAPIKeyError{Provider: "groq"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+path, body)
	if err != nil {
		return nil, ClassifyUpstreamError("groq", err)
	}
	req.ContentLength = contentLength
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ClassifyUpstreamError("groq", err)
	}

	return resp, nil
}
