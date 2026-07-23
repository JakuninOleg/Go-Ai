package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	RequestIDHeader = "X-Request-ID"
	DurationHeader  = "X-Go-Ai-Duration-Ms"
)

type requestIDContextKey struct{}

type Observer struct {
	Metrics *Metrics
	Logger  *slog.Logger
}

func New(logger *slog.Logger) *Observer {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	return &Observer{
		Metrics: NewMetrics(),
		Logger:  logger,
	}
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFromHeader(r.Header.Get(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func HTTPMetrics(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{
				ResponseWriter: w,
				start:          start,
			}

			next.ServeHTTP(recorder, r)

			metrics.RecordHTTP(recorder.Status())
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return ""
	}
	return requestID
}

func requestIDFromHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return ""
	}

	for _, r := range value {
		if r < 33 || r > 126 {
			return ""
		}
	}

	return value
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "req_" + strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", "")
	}

	return "req_" + hex.EncodeToString(bytes[:])
}

type responseRecorder struct {
	http.ResponseWriter
	start       time.Time
	status      int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}

	r.Header().Set(DurationHeader, durationMilliseconds(r.start))
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func (r *responseRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	flusher, ok := r.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func durationMilliseconds(start time.Time) string {
	return strconv.FormatInt(time.Since(start).Milliseconds(), 10)
}
