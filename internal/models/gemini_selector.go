package models

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var stableGeminiFlashID = regexp.MustCompile(`^gemini-([0-9]+)\.([0-9]+)-flash(?:-([0-9]+))?$`)

type RuntimeGeminiSelectionSnapshot struct {
	ActivePrimary      string    `json:"active_primary,omitempty"`
	LastKnownPrimary   string    `json:"last_known_primary,omitempty"`
	LastRefreshAt      time.Time `json:"last_refresh_at,omitempty"`
	LastSelectionAt    time.Time `json:"last_selection_at,omitempty"`
	LastResult         string    `json:"last_result"`
	LastResultCategory string    `json:"last_result_category"`
	ProcessLocal       bool      `json:"process_local"`
}

type RuntimeGeminiSelector struct {
	mu       sync.RWMutex
	snapshot RuntimeGeminiSelectionSnapshot
}

func NewRuntimeGeminiSelector() *RuntimeGeminiSelector {
	return &RuntimeGeminiSelector{
		snapshot: RuntimeGeminiSelectionSnapshot{
			LastResult:         "not_refreshed",
			LastResultCategory: "unavailable",
			ProcessLocal:       true,
		},
	}
}

func NormalizeGeminiModelID(modelID string) string {
	return strings.TrimPrefix(strings.TrimSpace(modelID), "models/")
}

func SelectStableGeminiFlashModel(modelIDs []string) string {
	candidates := make([]geminiFlashCandidate, 0, len(modelIDs))
	seen := make(map[string]struct{}, len(modelIDs))

	for _, modelID := range modelIDs {
		normalizedID := NormalizeGeminiModelID(modelID)
		if _, ok := seen[normalizedID]; ok {
			continue
		}
		seen[normalizedID] = struct{}{}

		matches := stableGeminiFlashID.FindStringSubmatch(normalizedID)
		if matches == nil {
			continue
		}

		major, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		minor, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}

		revision := 0
		if matches[3] != "" {
			revision, err = strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
		}

		candidates = append(candidates, geminiFlashCandidate{
			id:       normalizedID,
			major:    major,
			minor:    minor,
			revision: revision,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.major != right.major {
			return left.major > right.major
		}
		if left.minor != right.minor {
			return left.minor > right.minor
		}
		if left.revision != right.revision {
			return left.revision > right.revision
		}
		return left.id > right.id
	})

	return candidates[0].id
}

type geminiFlashCandidate struct {
	id       string
	major    int
	minor    int
	revision int
}

func (s *RuntimeGeminiSelector) ApplyCatalog(modelIDs []string, refreshedAt time.Time) {
	selected := SelectStableGeminiFlashModel(modelIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.LastRefreshAt = refreshedAt.UTC()
	if selected == "" {
		s.snapshot.ActivePrimary = ""
		s.snapshot.LastResult = "no_eligible_candidate"
		s.snapshot.LastResultCategory = "no_eligible_candidate"
		return
	}

	s.snapshot.ActivePrimary = selected
	s.snapshot.LastKnownPrimary = selected
	s.snapshot.LastSelectionAt = refreshedAt.UTC()
	s.snapshot.LastResult = "selected"
	s.snapshot.LastResultCategory = "eligible_candidate"
}

func (s *RuntimeGeminiSelector) RecordCatalogFailure(refreshedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.LastRefreshAt = refreshedAt.UTC()
	s.snapshot.LastResult = "catalog_error"
	s.snapshot.LastResultCategory = "catalog_error"
}

func (s *RuntimeGeminiSelector) ResolveCandidates(alias string) ([]ModelConfig, error) {
	staticCandidates, err := resolveStaticCandidates(alias)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	activePrimary := s.snapshot.ActivePrimary
	s.mu.RUnlock()

	switch alias {
	case DefaultModelAlias:
		if activePrimary == "" {
			return staticCandidates, nil
		}

		return append([]ModelConfig{{Name: activePrimary, Provider: ProviderGemini}}, staticCandidates...), nil
	case "gemini-flash":
		if activePrimary == "" {
			return nil, ModelUnavailableError{Alias: alias}
		}

		return []ModelConfig{{Name: activePrimary, Provider: ProviderGemini}}, nil
	default:
		return staticCandidates, nil
	}
}

func (s *RuntimeGeminiSelector) Aliases() map[string][]ModelConfig {
	aliases := staticAliases()

	s.mu.RLock()
	activePrimary := s.snapshot.ActivePrimary
	s.mu.RUnlock()

	if activePrimary == "" {
		return aliases
	}

	aliases[DefaultModelAlias] = append(
		[]ModelConfig{{Name: activePrimary, Provider: ProviderGemini}},
		aliases[DefaultModelAlias]...,
	)
	aliases["gemini-flash"] = []ModelConfig{{Name: activePrimary, Provider: ProviderGemini}}

	return aliases
}

func (s *RuntimeGeminiSelector) Snapshot() RuntimeGeminiSelectionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot
}
