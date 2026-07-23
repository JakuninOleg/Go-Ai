package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/jakuninoleg/Go-Ai/internal/observability"
)

func BearerAuth(sharedSecret string, observers ...*observability.Observer) func(http.Handler) http.Handler {
	secret := strings.TrimSpace(sharedSecret)
	observer := firstObserver(observers...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				recordAuthFailure(observer)
				writeJSONError(
					w,
					"API authentication is not configured",
					"server_error",
					"auth_not_configured",
					http.StatusServiceUnavailable,
				)
				return
			}

			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
				recordAuthFailure(observer)
				w.Header().Set("WWW-Authenticate", `Bearer realm="go-ai"`)
				writeJSONError(
					w,
					"missing or invalid bearer token",
					"authentication_error",
					"unauthorized",
					http.StatusUnauthorized,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func firstObserver(observers ...*observability.Observer) *observability.Observer {
	if len(observers) == 0 {
		return nil
	}
	return observers[0]
}

func recordAuthFailure(observer *observability.Observer) {
	if observer == nil || observer.Metrics == nil {
		return
	}
	observer.Metrics.RecordAuthFailure()
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], true
}
