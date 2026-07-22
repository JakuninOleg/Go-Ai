package providers

import (
	"context"
	"net/http"
)

type Provider interface {
	Chat(ctx context.Context, body []byte) (*http.Response, error)
}
