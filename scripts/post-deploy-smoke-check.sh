#!/usr/bin/env sh

# Calls configured external providers and may incur usage. It never prints
# credentials, response headers, or response bodies.
set -eu

: "${GO_AI_BASE_URL:?GO_AI_BASE_URL is required}"
: "${GO_AI_SHARED_SECRET:?GO_AI_SHARED_SECRET is required}"

base_url=${GO_AI_BASE_URL%/}

request() {
	name=$1
	body=$2

	status=$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' \
		--max-time "${GO_AI_SMOKE_TIMEOUT_SECONDS:-30}" \
		--request POST "$base_url/v1/chat/completions" \
		--header 'Content-Type: application/json' \
		--header "Authorization: Bearer $GO_AI_SHARED_SECRET" \
		--data "$body")

	case "$status" in
		2??) printf '%s status=%s\n' "$name" "$status" ;;
		*) printf '%s status=%s\n' "$name" "$status" >&2; return 1 ;;
	esac
}

health_status=$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' \
	--max-time "${GO_AI_SMOKE_TIMEOUT_SECONDS:-30}" "$base_url/health")
case "$health_status" in
	2??) printf 'health status=%s\n' "$health_status" ;;
	*) printf 'health status=%s\n' "$health_status" >&2; exit 1 ;;
esac

request nonstream '{"model":"gemini-flash","messages":[{"role":"user","content":"Reply with OK."}]}'

if [ "${GO_AI_SMOKE_STREAM:-0}" = "1" ]; then
	request stream '{"model":"gemini-flash","messages":[{"role":"user","content":"Reply with OK."}],"stream":true}'
fi
