package providers

import (
	"context"
	"fmt"
	"net/http"
)

type MissingAPIKeyError struct {
	Provider string
}

func (e MissingAPIKeyError) Error() string {
	return fmt.Sprintf("missing API key for provider: %s", e.Provider)
}

type Provider interface {
	Chat(ctx context.Context, body []byte) (*http.Response, error)
}

type ModelInfo struct {
	ID string `json:"id"`
}

type ModelLister interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
}
