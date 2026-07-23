# Go-Ai project context

Go-Ai is a small Go API layer between user applications and LLM providers. The current MVP exposes an OpenAI-compatible chat endpoint for clients and resolves local model aliases to provider-specific model names before proxying the request upstream.

## Architecture

- `cmd/api/main.go` wires configuration, providers, the provider router, services, routes, and HTTP middleware.
- `internal/routes` registers public and protected HTTP routes.
- `internal/handlers` owns HTTP request/response handling, JSON error responses, and API bearer authentication.
- `internal/services` owns OpenAI-style chat request parsing, model alias resolution, and provider selection.
- `internal/models` is the local model registry. Gemini is the default provider for MVP.
- `internal/providers` contains provider clients for Gemini and OpenRouter OpenAI-compatible endpoints.

## Current behavior

- Public route: `GET /health`.
- Protected route: `POST /v1/chat/completions` requires `Authorization: Bearer <GO_AI_SHARED_SECRET>`.
- If `model` is omitted, the service uses `models.DefaultModelAlias`.
- If `model` is present but unknown, the service returns a predictable `400` JSON error instead of silently falling back.
- Successful upstream responses are proxied with their status, headers, and body.
- Provider API keys must be configured before requests are sent upstream.

## Configuration

Environment variables are loaded from the process environment and local `.env` via `godotenv`:

- `PORT` defaults to `8080`.
- `GO_AI_SHARED_SECRET` protects chat completion requests. Keep it secret and never commit `.env`.
- `GEMINI_API_KEY`, `GEMINI_BASE_URL` for Gemini.
- `OPENROUTER_API_KEY`, `OPENROUTER_BASE_URL` for OpenRouter.

## Local development rules

- Do not log request bodies, provider keys, shared secrets, or `.env` values.
- Do not commit `.env` or other local secret files.
- Keep OpenAI-compatible request/response proxying intact for successful upstream responses.
- Tool-calling payload compatibility is supported through pass-through proxying only. Tool execution intentionally lives in calling applications/services for now; do not add executors or business-specific tool logic inside Go-Ai.
- Model aliases are a local contract for client apps. Update tests when changing alias behavior.
- Gemini/OpenRouter model slugs can change over time. Verify the model list against official provider documentation before relying on a slug for production.

## Validation

Run before committing Go changes:

```sh
gofmt -w <changed-go-files>
go test ./...
```
