package providers

import (
	"fmt"
	"strings"
)

type ProviderRouter struct {
	gemini     Provider
	openRouter Provider
}

func NewProviderRouter(
	gemini Provider,
	openRouter Provider,
) *ProviderRouter {

	return &ProviderRouter{
		gemini:     gemini,
		openRouter: openRouter,
	}
}

func (r *ProviderRouter) Resolve(
	providerName string,
) (Provider, error) {

	switch strings.ToLower(providerName) {

	case "openrouter":
		return r.openRouter, nil

	case "gemini":
		return r.gemini, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}
