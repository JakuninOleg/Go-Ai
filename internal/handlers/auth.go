package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func BearerAuth(sharedSecret string) func(http.Handler) http.Handler {
	secret := strings.TrimSpace(sharedSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
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

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], true
}
