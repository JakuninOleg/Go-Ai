package handlers

import (
	"mime"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/services"
)

func AudioTranscriptionHandler(service *services.AudioService, maxRequestBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength < 0 {
			writeJSONError(
				w,
				"Content-Length is required for audio transcription requests",
				"invalid_request_error",
				"content_length_required",
				http.StatusLengthRequired,
			)
			return
		}
		if r.ContentLength > maxRequestBytes {
			writeJSONError(
				w,
				"audio transcription request exceeds the configured byte limit",
				"invalid_request_error",
				"payload_too_large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}
		if !hasMediaType(r.Header.Get("Content-Type"), "multipart/form-data") {
			writeUnsupportedAudioContentType(w, "multipart/form-data")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		proxyAudioResponse(w, service, r, true)
	}
}

func AudioSpeechHandler(service *services.AudioService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !hasMediaType(r.Header.Get("Content-Type"), "application/json") {
			writeUnsupportedAudioContentType(w, "application/json")
			return
		}

		proxyAudioResponse(w, service, r, false)
	}
}

func proxyAudioResponse(w http.ResponseWriter, service *services.AudioService, r *http.Request, transcription bool) {
	var (
		resp *http.Response
		err  error
	)
	if transcription {
		resp, err = service.Transcribe(r.Context(), r.Header.Get("Content-Type"), r.ContentLength, r.Body)
	} else {
		resp, err = service.Speech(r.Context(), r.Header.Get("Content-Type"), r.ContentLength, r.Body)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_ = copyResponseBody(w, resp.Body)
}

func hasMediaType(contentType string, expected string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && strings.EqualFold(mediaType, expected)
}

func writeUnsupportedAudioContentType(w http.ResponseWriter, expected string) {
	writeJSONError(
		w,
		"Content-Type must be "+expected,
		"invalid_request_error",
		"unsupported_media_type",
		http.StatusUnsupportedMediaType,
	)
}
