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
