# Go-Ai

Go-Ai is a small Go HTTP API layer between client applications and LLM providers. It exposes an OpenAI-compatible chat completions endpoint and resolves local model aliases before proxying requests upstream.

## Configuration

The service reads configuration from environment variables and an optional local `.env` file:

- `PORT` defaults to `8080`.
- `GO_AI_SHARED_SECRET` is required for protected API requests.
- `GEMINI_API_KEY`, `GEMINI_BASE_URL` configure Gemini.
- `OPENROUTER_API_KEY`, `OPENROUTER_BASE_URL` configure OpenRouter.
- `MODEL_REFRESH_INTERVAL` controls provider model discovery refresh cadence and defaults to `1h`.

Do not commit `.env` or real secret values.

## Model aliases, fallback, and discovery

Client applications should send local aliases such as `default` or omit `model` entirely. They should not depend on real provider model slugs. Go-Ai rewrites the alias to the selected upstream model before proxying the request.

The `default` alias has ordered candidates. Go-Ai tries the primary Gemini model first and can fall back to a conservative OpenRouter free candidate when the upstream failure is retryable:

- provider/network error before a response is received;
- HTTP `429`, `500`, `502`, `503`, or `504` from the upstream provider.

Go-Ai does not fall back for invalid client requests, unknown aliases, missing provider API keys, or upstream `400`, `401`, and `403` responses. If every candidate fails, the gateway returns the final upstream response when one exists, or a gateway error for network failures. This improves resilience, but it cannot guarantee a response when all providers are down, quota is exhausted, auth is invalid, or the request itself is invalid.

Successful chat responses include diagnostic headers:

- `X-Go-Ai-Model-Alias`
- `X-Go-Ai-Provider`
- `X-Go-Ai-Upstream-Model`
- `X-Go-Ai-Fallback-Used`

For streaming requests, fallback is only possible before Go-Ai starts proxying the upstream response body. Once an SSE stream is being sent to the client, streams are not mixed or transparently replaced.

Go-Ai refreshes an in-memory provider model catalog on startup and then every hour by default. Discovery failures are logged as warnings and do not prevent the app from starting; the static alias registry remains the safe baseline. The protected `GET /v1/models` endpoint returns the configured aliases, candidates, discovered provider models, last successful refresh time, and safe discovery errors.

Redis is intentionally not required for the MVP. On the current Fly setup, an in-memory hourly catalog is enough for model discovery and avoids extra infrastructure. Redis can be added later if the service needs shared state across many instances, distributed rate limits, or stronger cross-instance cache coordination.

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

Inspect model routing status:

```sh
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>"
```

## Fly.io deployment

The production Fly app is `go-ai-i8r-lg` and is configured in `fly.toml` to build from the project `Dockerfile` and serve the API on port `8080`.

Runtime secrets stay in Fly, not in GitHub Actions. Configure them with Fly secrets before serving traffic:

- `GO_AI_SHARED_SECRET`
- `GEMINI_API_KEY`
- `OPENROUTER_API_KEY` if OpenRouter models are used

GitHub Actions deploys to Fly on pushes to `main` and can also be started manually from the Actions tab. To enable it, create a Fly deploy token for the app with `flyctl` or the Fly dashboard, then add it to the GitHub repository:

1. Create a deploy token scoped to `go-ai-i8r-lg`, for example with `fly tokens create deploy -a go-ai-i8r-lg`.
2. In GitHub, open repository Settings -> Secrets and variables -> Actions.
3. Add a new repository secret named `FLY_API_TOKEN` with the deploy token value.

Do not put provider API keys or `GO_AI_SHARED_SECRET` in the GitHub workflow; keep those values in Fly secrets.

## Tool calling compatibility

Go-Ai supports tool-calling payloads as an OpenAI-compatible proxy and model router. It does not execute tools itself: tool execution stays in client applications, such as Next apps.

For a Next-focused integration guide with production URL, auth, fetch examples, HTTP/SSE streaming, tool-calling flow, and voice-input guidance, see [docs/next-client.md](docs/next-client.md).

Streaming uses HTTP/SSE pass-through on `POST /v1/chat/completions` with `stream: true`; it is not a WebSocket flow. Keep `GO_AI_SHARED_SECRET` server-side and proxy browser streams through a trusted server route.

Flow:

1. The client sends `tools` and optional `tool_choice` to `POST /v1/chat/completions`.
2. Go-Ai resolves the local `model` alias to the provider model name and forwards the rest of the JSON request unchanged.
3. The provider response is proxied back unchanged, including assistant `tool_calls`.
4. The client executes the requested tool locally.
5. The client sends a follow-up chat request that includes the tool result as a message with `role: "tool"` and the matching `tool_call_id`.

Example request shape:

```json
{
  "model": "gemini-flash",
  "messages": [
    { "role": "user", "content": "What is the weather in Moscow?" }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather",
        "parameters": {
          "type": "object",
          "properties": { "city": { "type": "string" } },
          "required": ["city"]
        }
      }
    }
  ],
  "tool_choice": "auto"
}
```

## Render

`render.yaml` defines a Docker-based Render web service with `/health` as the health check path. Set secret environment variables in Render before serving traffic:

- `GO_AI_SHARED_SECRET`
- `GEMINI_API_KEY`
- `OPENROUTER_API_KEY` if OpenRouter models are used

Render provides `PORT` automatically for web services, and the API already listens on that value.
