package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
	"github.com/jakuninoleg/Go-Ai/internal/services"
)

type modelsResponse struct {
	DefaultAlias string                         `json:"default_alias"`
	Aliases      map[string][]modelCandidate    `json:"aliases"`
	Catalog      providers.ModelCatalogSnapshot `json:"catalog"`
}

type modelCandidate struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func ModelsHandler(service *services.AIService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aliases := service.ModelAliases()
		responseAliases := make(map[string][]modelCandidate, len(aliases))

		for alias, candidates := range aliases {
			responseCandidates := make([]modelCandidate, 0, len(candidates))
			for _, candidate := range candidates {
				responseCandidates = append(responseCandidates, modelCandidate{
					Provider: candidate.Provider,
					Model:    candidate.Name,
				})
			}
			responseAliases[alias] = responseCandidates
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(modelsResponse{
			DefaultAlias: models.DefaultModelAlias,
			Aliases:      responseAliases,
			Catalog:      service.ProviderModelCatalogSnapshot(),
		})
	}
}
