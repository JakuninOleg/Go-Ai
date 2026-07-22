package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

var ErrInvalidJSON = errors.New("invalid JSON request body")
var ErrInvalidRequestObject = errors.New("request body must be a JSON object")
var ErrModelMustBeString = errors.New("model must be a string")

type AIService struct {
	router *providers.ProviderRouter
}

func NewAIService(
	router *providers.ProviderRouter,
) *AIService {

	return &AIService{
		router: router,
	}
}

func (s *AIService) Chat(
	ctx context.Context,
	body []byte,
) (*http.Response, error) {

	var fields map[string]json.RawMessage

	err := json.Unmarshal(
		body,
		&fields,
	)

	if err != nil {
		return nil, ErrInvalidJSON
	}
	if fields == nil {
		return nil, ErrInvalidRequestObject
	}

	requestedModel := models.DefaultModelAlias

	if rawModel, ok := fields["model"]; ok {

		err = json.Unmarshal(
			rawModel,
			&requestedModel,
		)

		if err != nil {
			return nil, ErrModelMustBeString
		}
	}

	modelConfig, err := models.Resolve(requestedModel)
	if err != nil {
		return nil, err
	}

	newModel, err := json.Marshal(
		modelConfig.Name,
	)

	if err != nil {
		return nil, err
	}

	fields["model"] = newModel

	newBody, err := json.Marshal(fields)

	if err != nil {
		return nil, err
	}

	provider, err := s.router.Resolve(
		modelConfig.Provider,
	)
	if err != nil {
		return nil, err
	}

	return provider.Chat(
		ctx,
		newBody,
	)
}
