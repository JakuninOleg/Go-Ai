package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ProviderRouter struct {
	gemini     Provider
	openRouter Provider
	catalog    *ModelCatalog
}

type ProviderModelsSnapshot struct {
	Models    []ModelInfo `json:"models"`
	LastError string      `json:"last_error,omitempty"`
}

type ModelCatalogSnapshot struct {
	LastSuccessfulRefresh time.Time                         `json:"last_successful_refresh,omitempty"`
	Providers             map[string]ProviderModelsSnapshot `json:"providers"`
}

type ModelCatalog struct {
	mu                    sync.RWMutex
	modelsByProvider      map[string][]ModelInfo
	lastErrorByProvider   map[string]string
	lastSuccessfulRefresh time.Time
}

func NewProviderRouter(
	gemini Provider,
	openRouter Provider,
) *ProviderRouter {

	return &ProviderRouter{
		gemini:     gemini,
		openRouter: openRouter,
		catalog: &ModelCatalog{
			modelsByProvider:    make(map[string][]ModelInfo),
			lastErrorByProvider: make(map[string]string),
		},
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

func (r *ProviderRouter) RefreshModelCatalog(ctx context.Context) error {
	providers := map[string]Provider{
		"gemini":     r.gemini,
		"openrouter": r.openRouter,
	}

	refreshedModels := make(map[string][]ModelInfo)
	refreshErrors := make(map[string]string)
	var lastErr error

	for providerName, provider := range providers {
		lister, ok := provider.(ModelLister)
		if !ok {
			continue
		}

		models, err := lister.ListModels(ctx)
		if err != nil {
			refreshErrors[providerName] = err.Error()
			lastErr = err
			continue
		}

		modelsCopy := make([]ModelInfo, len(models))
		copy(modelsCopy, models)
		refreshedModels[providerName] = modelsCopy
	}

	r.catalog.applyRefresh(refreshedModels, refreshErrors)

	return lastErr
}

func (r *ProviderRouter) StartModelCatalogRefresh(ctx context.Context, interval time.Duration, logWarning func(error)) {
	if interval <= 0 {
		return
	}

	go func() {
		if err := r.RefreshModelCatalog(ctx); err != nil && logWarning != nil {
			logWarning(err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.RefreshModelCatalog(ctx); err != nil && logWarning != nil {
					logWarning(err)
				}
			}
		}
	}()
}

func (r *ProviderRouter) ModelCatalogSnapshot() ModelCatalogSnapshot {
	return r.catalog.snapshot()
}

func (r *ProviderRouter) IsKnownUnavailable(providerName string, modelID string) bool {
	return r.catalog.isKnownUnavailable(providerName, modelID)
}

func (c *ModelCatalog) applyRefresh(modelsByProvider map[string][]ModelInfo, lastErrorByProvider map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for providerName, models := range modelsByProvider {
		modelsCopy := make([]ModelInfo, len(models))
		copy(modelsCopy, models)
		c.modelsByProvider[providerName] = modelsCopy
		delete(c.lastErrorByProvider, providerName)
	}

	for providerName, lastError := range lastErrorByProvider {
		c.lastErrorByProvider[providerName] = lastError
	}

	if len(modelsByProvider) > 0 {
		c.lastSuccessfulRefresh = time.Now().UTC()
	}
}

func (c *ModelCatalog) snapshot() ModelCatalogSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	providers := make(map[string]ProviderModelsSnapshot, len(c.modelsByProvider)+len(c.lastErrorByProvider))

	for providerName, models := range c.modelsByProvider {
		modelsCopy := make([]ModelInfo, len(models))
		copy(modelsCopy, models)
		providers[providerName] = ProviderModelsSnapshot{
			Models: modelsCopy,
		}
	}

	for providerName, lastError := range c.lastErrorByProvider {
		snapshot := providers[providerName]
		snapshot.LastError = lastError
		providers[providerName] = snapshot
	}

	return ModelCatalogSnapshot{
		LastSuccessfulRefresh: c.lastSuccessfulRefresh,
		Providers:             providers,
	}
}

func (c *ModelCatalog) isKnownUnavailable(providerName string, modelID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	models, ok := c.modelsByProvider[strings.ToLower(providerName)]
	if !ok || len(models) == 0 {
		return false
	}

	for _, model := range models {
		if model.ID == modelID {
			return false
		}
	}

	return true
}
