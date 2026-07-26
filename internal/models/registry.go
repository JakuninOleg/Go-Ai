package models

import "fmt"

const (
	DefaultModelAlias = "default"

	ProviderGemini     = "gemini"
	ProviderOpenRouter = "openrouter"
)

type ModelConfig struct {
	Name     string
	Provider string
}

type AliasConfig struct {
	Candidates []ModelConfig
}

type UnknownModelError struct {
	Alias string
}

func (e UnknownModelError) Error() string {
	return fmt.Sprintf("unknown model: %s", e.Alias)
}

type ModelUnavailableError struct {
	Alias string
}

func (e ModelUnavailableError) Error() string {
	return fmt.Sprintf("model is currently unavailable: %s", e.Alias)
}

var aliasRegistry = map[string]AliasConfig{
	DefaultModelAlias: {
		Candidates: []ModelConfig{
			{
				Name:     "openrouter/free",
				Provider: ProviderOpenRouter,
			},
		},
	},

	"gemini-flash": {},

	"openrouter-gemini": {
		Candidates: []ModelConfig{
			{
				Name:     "google/gemini-2.5-flash",
				Provider: ProviderOpenRouter,
			},
		},
	},

	"openrouter-free": {
		Candidates: []ModelConfig{
			{
				Name:     "openrouter/free",
				Provider: ProviderOpenRouter,
			},
		},
	},
}

func resolveStaticCandidates(alias string) ([]ModelConfig, error) {
	aliasConfig, ok := aliasRegistry[alias]
	if !ok {
		return nil, UnknownModelError{Alias: alias}
	}

	candidates := make([]ModelConfig, len(aliasConfig.Candidates))
	copy(candidates, aliasConfig.Candidates)

	return candidates, nil
}

func staticAliases() map[string][]ModelConfig {
	aliases := make(map[string][]ModelConfig, len(aliasRegistry))
	for alias, config := range aliasRegistry {
		candidates := make([]ModelConfig, len(config.Candidates))
		copy(candidates, config.Candidates)
		aliases[alias] = candidates
	}

	return aliases
}
