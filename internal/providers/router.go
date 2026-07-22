package providers

import (
	"context"
	"net/http"
)

type ProviderRouter struct {
	primary  Provider
	fallback Provider
}

func NewProviderRouter(
	primary Provider,
	fallback Provider,
) *ProviderRouter {

	return &ProviderRouter{
		primary:  primary,
		fallback: fallback,
	}
}

func (r *ProviderRouter) Chat(
	ctx context.Context,
	body []byte,
) (*http.Response, error) {

	resp, err := r.primary.Chat(ctx, body)

	if err == nil && resp.StatusCode < 500 {
		return resp, nil
	}

	if resp != nil {
		resp.Body.Close()
	}

	return r.fallback.Chat(ctx, body)
}
