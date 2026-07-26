package services

import (
	"context"
	"io"
	"net/http"

	"github.com/jakuninoleg/Go-Ai/internal/providers"
)

type AudioService struct {
	provider providers.AudioProvider
}

func NewAudioService(provider providers.AudioProvider) *AudioService {
	return &AudioService{provider: provider}
}

func (s *AudioService) Transcribe(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return s.provider.Transcribe(ctx, contentType, contentLength, body)
}

func (s *AudioService) Speech(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return s.provider.Speech(ctx, contentType, contentLength, body)
}
