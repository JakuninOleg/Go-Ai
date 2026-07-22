package providers

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

type GeminiProvider struct {
	cfg config.APIConfig
}

func NewGeminiProvider(cfg config.APIConfig) *GeminiProvider {
	return &GeminiProvider{
		cfg: cfg,
	}
}

func (g *GeminiProvider) Chat(
	ctx context.Context,
	body []byte,
) (*http.Response, error) {
	if strings.TrimSpace(g.cfg.APIKey) == "" {
		return nil, MissingAPIKeyError{Provider: "gemini"}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.cfg.BaseURL+"/chat/completions",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+g.cfg.APIKey,
	)

	client := &http.Client{}

	return client.Do(req)
}
