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

type GeminiProvider struct {
	cfg    config.APIConfig
	client *http.Client
}

func NewGeminiProvider(cfg config.APIConfig) *GeminiProvider {
	return &GeminiProvider{
		cfg:    cfg,
		client: &http.Client{},
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

	return g.client.Do(req)
}

func (g *GeminiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if strings.TrimSpace(g.cfg.APIKey) == "" {
		return nil, MissingAPIKeyError{Provider: "gemini"}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		g.cfg.BaseURL+"/models",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+g.cfg.APIKey,
	)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("gemini model discovery failed with status %d", resp.StatusCode)
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
