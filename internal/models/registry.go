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

var Registry = map[string]ModelConfig{
	DefaultModelAlias:   AliasRegistry[DefaultModelAlias].Candidates[0],
	"gemini-flash":      AliasRegistry["gemini-flash"].Candidates[0],
	"openrouter-gemini": AliasRegistry["openrouter-gemini"].Candidates[0],
	"openrouter-free":   AliasRegistry["openrouter-free"].Candidates[0],
}

var AliasRegistry = map[string]AliasConfig{
	DefaultModelAlias: {
		Candidates: []ModelConfig{
			{
				Name:     "gemini-3.5-flash",
				Provider: ProviderGemini,
			},
			{
				Name:     "google/gemini-2.0-flash-exp:free",
				Provider: ProviderOpenRouter,
			},
		},
	},

	"gemini-flash": {
		Candidates: []ModelConfig{
			{
				Name:     "gemini-3.5-flash",
				Provider: ProviderGemini,
			},
		},
	},

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
				Name:     "google/gemini-2.0-flash-exp:free",
				Provider: ProviderOpenRouter,
			},
		},
	},
}

func Resolve(alias string) (ModelConfig, error) {
	modelConfig, ok := Registry[alias]
	if !ok {
		return ModelConfig{}, UnknownModelError{Alias: alias}
	}

	return modelConfig, nil
}

func ResolveCandidates(alias string) ([]ModelConfig, error) {
	aliasConfig, ok := AliasRegistry[alias]
	if !ok || len(aliasConfig.Candidates) == 0 {
		return nil, UnknownModelError{Alias: alias}
	}

	candidates := make([]ModelConfig, len(aliasConfig.Candidates))
	copy(candidates, aliasConfig.Candidates)

	return candidates, nil
}
