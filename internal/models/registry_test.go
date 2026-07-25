package models

import "testing"

func TestResolveReturnsDefaultModel(t *testing.T) {
	modelConfig, err := Resolve(DefaultModelAlias)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if modelConfig.Provider != ProviderGemini {
		t.Fatalf("expected default provider %q, got %q", ProviderGemini, modelConfig.Provider)
	}
	if modelConfig.Name == "" {
		t.Fatal("expected default model name to be set")
	}
}

func TestResolveReturnsUnknownModelError(t *testing.T) {
	_, err := Resolve("missing-model")
	if err == nil {
		t.Fatal("expected error")
	}

	unknownModelErr, ok := err.(UnknownModelError)
	if !ok {
		t.Fatalf("expected UnknownModelError, got %T", err)
	}
	if unknownModelErr.Alias != "missing-model" {
		t.Fatalf("expected alias %q, got %q", "missing-model", unknownModelErr.Alias)
	}
}

func TestAliasRegistryUsesFreeDefaultAndConfirmedExplicitGemini(t *testing.T) {
	testCases := map[string][]ModelConfig{
		DefaultModelAlias: {
			{Name: "gemini-3.6-flash", Provider: ProviderGemini},
			{Name: "openrouter/free", Provider: ProviderOpenRouter},
		},
		"gemini-flash": {
			{Name: "gemini-3.6-flash", Provider: ProviderGemini},
		},
		"openrouter-gemini": {
			{Name: "google/gemini-2.5-flash", Provider: ProviderOpenRouter},
		},
		"openrouter-free": {
			{Name: "openrouter/free", Provider: ProviderOpenRouter},
		},
	}

	for alias, expected := range testCases {
		candidates, err := ResolveCandidates(alias)
		if err != nil {
			t.Fatalf("ResolveCandidates(%q) returned error: %v", alias, err)
		}

		if len(candidates) != len(expected) {
			t.Fatalf("ResolveCandidates(%q) returned %#v, expected %#v", alias, candidates, expected)
		}
		for index, expectedCandidate := range expected {
			if candidates[index] != expectedCandidate {
				t.Fatalf("ResolveCandidates(%q)[%d] = %#v, expected %#v", alias, index, candidates[index], expectedCandidate)
			}
		}
	}
}

func TestDefaultFallsBackOnlyToFreeOpenRouterRoute(t *testing.T) {
	candidates, err := ResolveCandidates(DefaultModelAlias)
	if err != nil {
		t.Fatalf("ResolveCandidates(%q) returned error: %v", DefaultModelAlias, err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected confirmed Gemini plus free OpenRouter default candidates, got %#v", candidates)
	}
	if candidates[0] != (ModelConfig{Name: "gemini-3.6-flash", Provider: ProviderGemini}) {
		t.Fatalf("unexpected default primary candidate: %#v", candidates[0])
	}
	if candidates[1] != (ModelConfig{Name: "openrouter/free", Provider: ProviderOpenRouter}) {
		t.Fatalf("unexpected default fallback candidate: %#v", candidates[1])
	}
}
