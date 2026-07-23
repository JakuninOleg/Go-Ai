package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/config"
)

type OpenRouterProvider struct {
	cfg    config.APIConfig
	client *http.Client
}

func NewOpenRouterProvider(cfg config.APIConfig) *OpenRouterProvider {
	return &OpenRouterProvider{
		cfg:    cfg,
		client: &http.Client{},
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

	return p.client.Do(req)
}

func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return nil, MissingAPIKeyError{Provider: "openrouter"}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.cfg.BaseURL+"/models",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+p.cfg.APIKey,
	)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openrouter model discovery failed with status %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(payload.Data))
	for _, model := range payload.Data {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}
		models = append(models, ModelInfo{ID: model.ID})
	}

	return models, nil
}
