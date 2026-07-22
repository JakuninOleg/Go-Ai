package providers

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

type OpenRouterProvider struct {
	cfg config.APIConfig
}

func NewOpenRouterProvider(cfg config.APIConfig) *OpenRouterProvider {
	return &OpenRouterProvider{
		cfg: cfg,
	}
}

func (p *OpenRouterProvider) Chat(
	ctx context.Context,
	body []byte,
) (*http.Response, error) {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return nil, MissingAPIKeyError{Provider: "openrouter"}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.cfg.BaseURL+"/chat/completions",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set(
		"Authorization",
		"Bearer "+p.cfg.APIKey,
	)

	client := &http.Client{}

	return client.Do(req)
}
