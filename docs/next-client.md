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

## Streaming responses

Go-Ai supports streaming as HTTP/SSE pass-through on the same endpoint: `POST /v1/chat/completions`. This is not a WebSocket flow. Send `stream: true` in the OpenAI-compatible request body and read the response as a stream.

Go-Ai does not parse or modify SSE chunks. It resolves the local model alias, forwards the request upstream, then proxies the upstream status, headers, and body back to the caller. Streaming tool calls may arrive split across multiple SSE chunks, so your Next app or browser UI must assemble partial deltas before executing or displaying structured tool-call data. Tool execution still stays in the client/Next application; Go-Ai only passes payloads through.

Quick deployed smoke test with curl:

```sh
curl -N https://go-ai-i8r-lg.fly.dev/v1/chat/completions \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"gemini-flash",
    "messages":[{"role":"user","content":"Say hello in one short sentence."}],
    "stream":true
  }'
```

Use `-N` so curl does not buffer the streamed response.

### Next server route proxy

Keep `GO_AI_SHARED_SECRET` only on the server side of your Next app. Browser code must not call Go-Ai directly with the shared secret. Instead, proxy the stream through a Next route handler:

```ts
// app/api/chat/stream/route.ts
import { NextRequest } from "next/server";

const GO_AI_BASE_URL = "https://go-ai-i8r-lg.fly.dev";

export async function POST(request: NextRequest) {
  const { messages } = await request.json();

  const upstream = await fetch(`${GO_AI_BASE_URL}/v1/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${process.env.GO_AI_SHARED_SECRET}`,
    },
    body: JSON.stringify({
      model: "gemini-flash",
      messages,
      stream: true,
    }),
  });

  if (!upstream.body) {
    return new Response(await upstream.text(), {
      status: upstream.status,
      headers: {
        "Content-Type": upstream.headers.get("Content-Type") ?? "application/json",
      },
    });
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("Content-Type") ?? "text/event-stream",
      "Cache-Control": upstream.headers.get("Cache-Control") ?? "no-cache",
    },
  });
}
```

Do not call `await res.json()` for streaming responses. Read `response.body` as a `ReadableStream` instead.

### Browser ReadableStream example

The browser calls your Next route, not Go-Ai directly:

```ts
const response = await fetch("/api/chat/stream", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    messages: [{ role: "user", content: "Say hello." }],
  }),
});

if (!response.body) {
  throw new Error("Streaming is not available in this browser.");
}

const reader = response.body.getReader();
const decoder = new TextDecoder();

let sseBuffer = "";

while (true) {
  const { value, done } = await reader.read();
  if (done) break;

  sseBuffer += decoder.decode(value, { stream: true });

  const events = sseBuffer.split("\n\n");
  sseBuffer = events.pop() ?? "";

  for (const event of events) {
    if (!event.startsWith("data:")) continue;

    const data = event.slice("data:".length).trim();
    if (data === "[DONE]") return;

    // Parse provider/OpenAI-compatible deltas here and update your UI.
    // Tool-call deltas may be partial and need accumulation before use.
    console.log(data);
  }
}
```

For a typing effect in the UI, prefer real provider-to-Next-to-browser streaming for long responses. A client-side typing simulation after receiving the full response is simpler, but it does not reduce time to first token.

## Voice input

Vercel AI SDK is not required for voice input itself. Voice input is usually implemented with one of these approaches:

- browser Web Speech API;
- browser `MediaRecorder` plus Whisper or another speech-to-text provider;
- an external STT service.

After transcription, send the resulting text to Go-Ai as a normal chat message. Vercel AI SDK may still help with chat state or streaming UI, but it is optional for voice capture/transcription.
