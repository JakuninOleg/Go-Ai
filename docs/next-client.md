# Next client integration guide

This guide is intended for Next applications and agents that use Go-Ai as an OpenAI-compatible chat proxy.

## Production API

Production base URL:

```text
https://go-ai-i8r-lg.fly.dev
```

Use placeholders for secrets in code and documentation. Never commit real values.

## Health check

`GET /health` is public and does not require authorization.

```sh
curl https://go-ai-i8r-lg.fly.dev/health
```

## Chat completions

`POST /v1/chat/completions` requires bearer authentication:

```http
Authorization: Bearer <GO_AI_SHARED_SECRET>
```

Keep `GO_AI_SHARED_SECRET` only on the server side of your Next app: route handlers, server actions, backend jobs, or other trusted server code. Do not expose it in browser code and do not put it in `NEXT_PUBLIC_*` variables.

## Minimal Next / TypeScript fetch example

Example route handler without Vercel AI SDK:

```ts
// app/api/chat/route.ts
import { NextRequest } from "next/server";

const GO_AI_BASE_URL = "https://go-ai-i8r-lg.fly.dev";

export async function POST(request: NextRequest) {
  const { messages } = await request.json();

  const response = await fetch(`${GO_AI_BASE_URL}/v1/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${process.env.GO_AI_SHARED_SECRET}`,
    },
    body: JSON.stringify({
      messages,
      // If model is omitted, Go-Ai uses the default Gemini alias.
    }),
  });

  const body = await response.text();

  return new Response(body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "application/json",
    },
  });
}
```

Request body with the default Gemini model:

```json
{
  "messages": [
    { "role": "user", "content": "Say hello in one sentence." }
  ]
}
```

Request body with an explicit local model alias:

```json
{
  "model": "gemini-flash",
  "messages": [
    { "role": "user", "content": "Say hello in one sentence." }
  ]
}
```

Go-Ai resolves local aliases to provider-specific model names before forwarding the request upstream.

## Tool calling: Variant A flow

Go-Ai does not execute tools and does not know your application's database, APIs, permissions, or business logic. It only proxies OpenAI-compatible tool-calling payloads.

The client or Next server owns tool execution:

1. Next sends a chat request with `tools` and optional `tool_choice` / `parallel_tool_calls`.
2. Go-Ai resolves the local model alias and forwards the JSON payload to the provider.
3. The model may return assistant `tool_calls`.
4. Next validates and executes the requested tools in application code.
5. Next sends a follow-up chat request that includes each tool result as a message with `role: "tool"` and the matching `tool_call_id`.

Initial request example:

```json
{
  "messages": [
    { "role": "user", "content": "What is the weather in Moscow?" }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather for a city",
        "parameters": {
          "type": "object",
          "properties": {
            "city": { "type": "string" }
          },
          "required": ["city"]
        }
      }
    }
  ],
  "tool_choice": "auto",
  "parallel_tool_calls": true
}
```

Follow-up request after the model asks for a tool:

```json
{
  "model": "gemini-flash",
  "messages": [
    { "role": "user", "content": "What is the weather in Moscow?" },
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "id": "call_weather_1",
          "type": "function",
          "function": {
            "name": "get_weather",
            "arguments": "{\"city\":\"Moscow\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_weather_1",
      "content": "{\"temperature\":\"-5 C\",\"condition\":\"snow\"}"
    }
  ]
}
```

## Streaming and gradual typing in UI

Go-Ai is designed to preserve proxying behavior. If the upstream provider and your client use `stream: true`, a Next route can read the upstream `ReadableStream` and return Server-Sent Events or another stream format to the UI. Vercel AI SDK is not required for this, although it can be useful for chat UI helpers.

Current status: streaming is a pass-through goal, but it should be validated end-to-end with the deployed provider/client path before depending on it in production.

For a typing effect in the UI, you have two options:

- real streaming from provider to Next to browser, which is the better UX for long responses;
- client-side typing simulation after receiving a full response, which is simpler but does not reduce perceived wait time before the first token.

## Voice input

Vercel AI SDK is not required for voice input itself. Voice input is usually implemented with one of these approaches:

- browser Web Speech API;
- browser `MediaRecorder` plus Whisper or another speech-to-text provider;
- an external STT service.

After transcription, send the resulting text to Go-Ai as a normal chat message. Vercel AI SDK may still help with chat state or streaming UI, but it is optional for voice capture/transcription.
