package models

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSelectStableGeminiFlashModelNormalizesFiltersAndRanks(t *testing.T) {
	selected := SelectStableGeminiFlashModel([]string{
		"models/gemini-2.5-flash",
		"gemini-3.7-flash",
		"gemini-3.10-flash",
		"gemini-3.10-flash-001",
		"gemini-3.10-flash-002",
		"gemini-4.0-pro",
		"gemini-4.0-flash-preview",
		"gemini-4.0-flash-exp",
		"gemini-4.0-flash-lite",
		"gemini-4.0-flash-image",
		"gemini-4.0-flash-audio",
		"gemini-4.0-flash-live",
		"gemini-4.0-flash-tts",
		"gemini-4.0-flash-embedding",
		"gemini-4.0-flash-research",
		"gemini-4.0-flash-computer",
		"gemini-4.0-flash-robotics",
		"gemini-4.0-flash-native",
		"gemini-4.0-flash-translate",
		"gemini-4.0-flash-generate",
		"gemini-4.0-flash-omni",
		"gemini-flash-latest",
	})

	if selected != "gemini-3.10-flash-002" {
		t.Fatalf("expected highest stable Flash model, got %q", selected)
	}
}

func TestSelectStableGeminiFlashModelReturnsEmptyWithoutEligibleCandidate(t *testing.T) {
	selected := SelectStableGeminiFlashModel([]string{
		"gemini-3.7-pro",
		"gemini-3.7-flash-preview",
		"gemini-flash-latest",
	})
	if selected != "" {
		t.Fatalf("expected no selected model, got %q", selected)
	}
}

func TestRuntimeGeminiSelectorDefaultUsesOpenRouterUntilCatalogSelectsGemini(t *testing.T) {
	selector := NewRuntimeGeminiSelector()

	candidates, err := selector.ResolveCandidates(DefaultModelAlias)
	if err != nil {
		t.Fatalf("ResolveCandidates returned error: %v", err)
	}
	assertCandidates(t, candidates, []ModelConfig{{Name: "openrouter/free", Provider: ProviderOpenRouter}})

	_, err = selector.ResolveCandidates("gemini-flash")
	var unavailableErr ModelUnavailableError
	if !errors.As(err, &unavailableErr) {
		t.Fatalf("expected ModelUnavailableError, got %v", err)
	}

	selector.ApplyCatalog([]string{"models/gemini-3.7-flash"}, time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC))
	candidates, err = selector.ResolveCandidates(DefaultModelAlias)
	if err != nil {
		t.Fatalf("ResolveCandidates returned error: %v", err)
	}
	assertCandidates(t, candidates, []ModelConfig{
		{Name: "gemini-3.7-flash", Provider: ProviderGemini},
		{Name: "openrouter/free", Provider: ProviderOpenRouter},
	})
}

func TestRuntimeGeminiSelectorRetainsActivePrimaryAfterCatalogFailure(t *testing.T) {
	selector := NewRuntimeGeminiSelector()
	selector.ApplyCatalog([]string{"gemini-3.7-flash"}, time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC))
	selector.RecordCatalogFailure(time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC))

	snapshot := selector.Snapshot()
	if snapshot.ActivePrimary != "gemini-3.7-flash" || snapshot.LastKnownPrimary != "gemini-3.7-flash" {
		t.Fatalf("expected selected primary to remain active after catalog failure, got %#v", snapshot)
	}
	if snapshot.LastResultCategory != "catalog_error" {
		t.Fatalf("expected catalog error category, got %q", snapshot.LastResultCategory)
	}
}

func TestRuntimeGeminiSelectorUpdatesToNewerCatalogCandidate(t *testing.T) {
	selector := NewRuntimeGeminiSelector()
	selector.ApplyCatalog([]string{"gemini-3.7-flash"}, time.Now())
	selector.ApplyCatalog([]string{"gemini-3.7-flash", "gemini-3.10-flash"}, time.Now())

	snapshot := selector.Snapshot()
	if snapshot.ActivePrimary != "gemini-3.10-flash" {
		t.Fatalf("expected newer primary, got %#v", snapshot)
	}
}

func TestRuntimeGeminiSelectorSupportsConcurrentRefreshAndResolution(t *testing.T) {
	selector := NewRuntimeGeminiSelector()
	var waitGroup sync.WaitGroup

	for index := 0; index < 16; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			if index%2 == 0 {
				selector.ApplyCatalog([]string{"gemini-3.7-flash", "gemini-3.10-flash"}, time.Now())
				return
			}
			_, _ = selector.ResolveCandidates(DefaultModelAlias)
			_ = selector.Snapshot()
		}(index)
	}

	waitGroup.Wait()
}

func TestRuntimeGeminiSelectorPreservesStaticExplicitAliases(t *testing.T) {
	selector := NewRuntimeGeminiSelector()

	for alias, expected := range map[string][]ModelConfig{
		"openrouter-gemini": {{Name: "google/gemini-2.5-flash", Provider: ProviderOpenRouter}},
		"openrouter-free":   {{Name: "openrouter/free", Provider: ProviderOpenRouter}},
	} {
		candidates, err := selector.ResolveCandidates(alias)
		if err != nil {
			t.Fatalf("ResolveCandidates(%q) returned error: %v", alias, err)
		}
		assertCandidates(t, candidates, expected)
	}
}

func TestRuntimeGeminiSelectorReturnsUnknownModelError(t *testing.T) {
	_, err := NewRuntimeGeminiSelector().ResolveCandidates("missing-model")
	var unknownErr UnknownModelError
	if !errors.As(err, &unknownErr) || unknownErr.Alias != "missing-model" {
		t.Fatalf("expected UnknownModelError for missing alias, got %v", err)
	}
}

func assertCandidates(t *testing.T, actual, expected []ModelConfig) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
	for index, expectedCandidate := range expected {
		if actual[index] != expectedCandidate {
			t.Fatalf("candidate %d: expected %#v, got %#v", index, expectedCandidate, actual[index])
		}
	}
}
