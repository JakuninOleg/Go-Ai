package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jakuninoleg/Go-Ai/internal/services"
)

type audioCaptureProvider struct {
	transcribe func(context.Context, string, int64, io.ReadCloser) (*http.Response, error)
	speech     func(context.Context, string, int64, io.ReadCloser) (*http.Response, error)
}

func (p *audioCaptureProvider) Transcribe(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return p.transcribe(ctx, contentType, contentLength, body)
}

func (p *audioCaptureProvider) Speech(
	ctx context.Context,
	contentType string,
	contentLength int64,
	body io.ReadCloser,
) (*http.Response, error) {
	return p.speech(ctx, contentType, contentLength, body)
}

func TestAudioTranscriptionHandlerRequiresContentLength(t *testing.T) {
	called := false
	service := services.NewAudioService(&audioCaptureProvider{
		transcribe: func(context.Context, string, int64, io.ReadCloser) (*http.Response, error) {
			called = true
			return nil, nil
		},
	})
	handler := AudioTranscriptionHandler(service, 25_000_000)

	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader([]byte("body")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	req.ContentLength = -1
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusLengthRequired {
		t.Fatalf("expected status %d, got %d", http.StatusLengthRequired, response.Code)
	}
	if called {
		t.Fatal("expected transcription provider not to be called")
	}
	assertAudioErrorCode(t, response, "content_length_required")
}

func TestAudioTranscriptionHandlerRejectsRequestOverByteLimit(t *testing.T) {
	called := false
	service := services.NewAudioService(&audioCaptureProvider{
		transcribe: func(context.Context, string, int64, io.ReadCloser) (*http.Response, error) {
			called = true
			return nil, nil
		},
	})
	handler := AudioTranscriptionHandler(service, 10)

	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader([]byte("body")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	req.ContentLength = 11
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
	if called {
		t.Fatal("expected transcription provider not to be called")
	}
	assertAudioErrorCode(t, response, "payload_too_large")
}

func TestAudioTranscriptionHandlerStreamsMultipartBodyAndResponse(t *testing.T) {
	requestBody := []byte("--test\r\nContent-Disposition: form-data; name=\"model\"\r\n\r\nwhisper-large-v3-turbo\r\n--test--\r\n")
	service := services.NewAudioService(&audioCaptureProvider{
		transcribe: func(_ context.Context, contentType string, contentLength int64, body io.ReadCloser) (*http.Response, error) {
			if contentType != "multipart/form-data; boundary=test" {
				t.Fatalf("unexpected content type: %q", contentType)
			}
			if contentLength != int64(len(requestBody)) {
				t.Fatalf("unexpected content length: %d", contentLength)
			}
			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("failed to read forwarded body: %v", err)
			}
			if !bytes.Equal(got, requestBody) {
				t.Fatalf("unexpected forwarded body: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"text":"hello"}`))),
			}, nil
		},
	})
	handler := AudioTranscriptionHandler(service, 25_000_000)

	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response content type: %q", response.Header().Get("Content-Type"))
	}
	if got := response.Body.String(); got != `{"text":"hello"}` {
		t.Fatalf("unexpected response body: %q", got)
	}
}

func TestAudioSpeechHandlerProxiesJSONRequestAndBinaryResponse(t *testing.T) {
	requestBody := []byte(`{"model":"canopylabs/orpheus-v1-english","input":"Hello","voice":"austin"}`)
	responseBody := []byte{0x52, 0x49, 0x46, 0x46}
	service := services.NewAudioService(&audioCaptureProvider{
		speech: func(_ context.Context, contentType string, contentLength int64, body io.ReadCloser) (*http.Response, error) {
			if contentType != "application/json" {
				t.Fatalf("unexpected content type: %q", contentType)
			}
			if contentLength != int64(len(requestBody)) {
				t.Fatalf("unexpected content length: %d", contentLength)
			}
			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("failed to read forwarded body: %v", err)
			}
			if !bytes.Equal(got, requestBody) {
				t.Fatalf("unexpected forwarded body: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Content-Type": []string{"audio/wav"}},
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
			}, nil
		},
	})
	handler := AudioSpeechHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if response.Header().Get("Content-Type") != "audio/wav" {
		t.Fatalf("unexpected response content type: %q", response.Header().Get("Content-Type"))
	}
	if !bytes.Equal(response.Body.Bytes(), responseBody) {
		t.Fatalf("unexpected binary response: %v", response.Body.Bytes())
	}
}

func assertAudioErrorCode(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var payload errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Error.Code != want {
		t.Fatalf("expected error code %q, got %q", want, payload.Error.Code)
	}
}
