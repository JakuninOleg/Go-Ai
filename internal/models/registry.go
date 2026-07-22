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

type UnknownModelError struct {
	Alias string
}

func (e UnknownModelError) Error() string {
	return fmt.Sprintf("unknown model: %s", e.Alias)
}

var Registry = map[string]ModelConfig{
	DefaultModelAlias: {
		Name:     "gemini-3.5-flash",
		Provider: ProviderGemini,
	},

	"gemini-flash": {
		Name:     "gemini-3.5-flash",
		Provider: ProviderGemini,
	},

	"openrouter-gemini": {
		Name:     "google/gemini-2.5-flash",
		Provider: ProviderOpenRouter,
	},

	"openrouter-free": {
		Name:     "google/gemini-2.0-flash-exp:free",
		Provider: ProviderOpenRouter,
	},
}

func Resolve(alias string) (ModelConfig, error) {
	modelConfig, ok := Registry[alias]
	if !ok {
		return ModelConfig{}, UnknownModelError{Alias: alias}
	}

	return modelConfig, nil
}
