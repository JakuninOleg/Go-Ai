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

## Tool calling compatibility

Go-Ai supports tool-calling payloads as an OpenAI-compatible proxy and model router. It does not execute tools itself: tool execution stays in client applications, such as Next apps.

For a Next-focused integration guide with production URL, auth, fetch examples, tool-calling flow, streaming notes, and voice-input guidance, see [docs/next-client.md](docs/next-client.md).

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
