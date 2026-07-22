# Go-Ai

Go-Ai is a small Go HTTP API layer between client applications and LLM providers. It exposes an OpenAI-compatible chat completions endpoint and resolves local model aliases before proxying requests upstream.

## Configuration

The service reads configuration from environment variables and an optional local `.env` file:

- `PORT` defaults to `8080`.
- `GO_AI_SHARED_SECRET` is required for protected API requests.
- `GEMINI_API_KEY`, `GEMINI_BASE_URL` configure Gemini.
- `OPENROUTER_API_KEY`, `OPENROUTER_BASE_URL` configure OpenRouter.

Do not commit `.env` or real secret values.

## Docker

Build the image:

```sh
docker build -t go-ai:local .
```

Run the API locally:

```sh
docker run --rm \
  -p 8080:8080 \
  -e PORT=8080 \
  -e GO_AI_SHARED_SECRET=change-me \
  -e GEMINI_API_KEY=your-gemini-key \
  go-ai:local
```

Health check:

```sh
curl http://localhost:8080/health
```

## Render

`render.yaml` defines a Docker-based Render web service with `/health` as the health check path. Set secret environment variables in Render before serving traffic:

- `GO_AI_SHARED_SECRET`
- `GEMINI_API_KEY`
- `OPENROUTER_API_KEY` if OpenRouter models are used

Render provides `PORT` automatically for web services, and the API already listens on that value.
