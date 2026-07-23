package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jakuninoleg/Go-Ai/internal/models"
	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

var ErrInvalidJSON = errors.New("invalid JSON request body")
var ErrInvalidRequestObject = errors.New("request body must be a JSON object")
var ErrModelMustBeString = errors.New("model must be a string")
var ErrNoAvailableModelCandidate = errors.New("no available model candidate")

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

	candidates, err := models.ResolveCandidates(requestedModel)
	if err != nil {
		return nil, err
	}

	var lastResponse *http.Response
	var lastErr error

	for index, candidate := range candidates {
		if s.router.IsKnownUnavailable(candidate.Provider, candidate.Name) {
			lastErr = ErrNoAvailableModelCandidate
			continue
		}

		newBody, err := bodyForModel(
			fields,
			candidate.Name,
		)
		if err != nil {
			return nil, err
		}

		provider, err := s.router.Resolve(
			candidate.Provider,
		)
		if err != nil {
			return nil, err
		}

		resp, err := provider.Chat(
			ctx,
			newBody,
		)
		if err != nil {
			if !isRetryableProviderError(err) || index == len(candidates)-1 {
				closeResponseBody(lastResponse)
				return nil, err
			}

			lastErr = err
			continue
		}
		if resp.Header == nil {
			resp.Header = make(http.Header)
		}

		addDiagnosticHeaders(
			resp.Header,
			requestedModel,
			candidate,
			index > 0,
		)

		if !isRetryableUpstreamStatus(resp.StatusCode) || index == len(candidates)-1 {
			closeResponseBody(lastResponse)
			return resp, nil
		}

		closeResponseBody(lastResponse)
		lastResponse = resp
	}

	if lastResponse != nil {
		return lastResponse, nil
	}

	return nil, lastErr
}

func (s *AIService) ModelAliases() map[string][]models.ModelConfig {
	aliases := make(map[string][]models.ModelConfig, len(models.AliasRegistry))

	for alias, config := range models.AliasRegistry {
		candidates := make([]models.ModelConfig, len(config.Candidates))
		copy(candidates, config.Candidates)
		aliases[alias] = candidates
	}

	return aliases
}

func (s *AIService) ProviderModelCatalogSnapshot() providers.ModelCatalogSnapshot {
	return s.router.ModelCatalogSnapshot()
}

func (s *AIService) RefreshProviderModelCatalog(ctx context.Context) error {
	return s.router.RefreshModelCatalog(ctx)
}

func (s *AIService) StartProviderModelCatalogRefresh(ctx context.Context, interval time.Duration, logWarning func(error)) {
	s.router.StartModelCatalogRefresh(ctx, interval, logWarning)
}

func bodyForModel(
	fields map[string]json.RawMessage,
	modelName string,
) ([]byte, error) {
	newFields := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		newFields[key] = value
	}

	newModel, err := json.Marshal(modelName)
	if err != nil {
		return nil, err
	}

	newFields["model"] = newModel

	return json.Marshal(newFields)
}

func isRetryableProviderError(err error) bool {
	var missingAPIKeyErr providers.MissingAPIKeyError
	if errors.As(err, &missingAPIKeyErr) {
		return false
	}

	return true
}

func isRetryableUpstreamStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func addDiagnosticHeaders(
	header http.Header,
	alias string,
	candidate models.ModelConfig,
	fallbackUsed bool,
) {
	header.Set("X-Go-Ai-Model-Alias", alias)
	header.Set("X-Go-Ai-Provider", candidate.Provider)
	header.Set("X-Go-Ai-Upstream-Model", candidate.Name)
	header.Set("X-Go-Ai-Fallback-Used", strconv.FormatBool(fallbackUsed))
}

func closeResponseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
