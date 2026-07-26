# Next client integration guide

This is the Next.js integration guide. Go-Ai itself is a generic HTTP/OpenAI-compatible gateway; any trusted backend or service can call it over HTTP. The examples below focus on keeping Go-Ai secrets server-side in Next route handlers and other Next server code.

## Production API

Production base URL:

```text
https://go-ai-i8r-lg.fly.dev
```

Use placeholders for secrets in code and documentation. Never commit real values.

## Health check

`GET /health` is public and does not require authorization. It is a simple liveness check; it does not verify provider keys, upstream providers, or the model catalog.

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
      // If model is omitted, Go-Ai uses the default local alias.
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

Request body with the default local model alias:

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
  "model": "default",
  "messages": [
    { "role": "user", "content": "Say hello in one sentence." }
  ]
}
```

Go-Ai resolves local aliases to provider-specific model names before forwarding the request upstream. Next applications should keep using aliases such as `default` or omit `model`; they should not know or store real provider model slugs.

## Model fallback and catalog diagnostics

At startup and each catalog refresh, `default` selects the highest normalized Gemini ID exactly matching stable `gemini-<major>.<minor>-flash` with an optional numeric revision. It can fall back on retryable upstream failures to OpenRouter's `openrouter/free` router; with no eligible direct Gemini candidate, it uses OpenRouter only. `gemini-flash` requires that direct runtime primary and returns `503 model_unavailable` if unavailable. This is catalog-only promotion, not a guarantee that a candidate supports paid use, SSE, tools, or an application's specific workflow. `openrouter-gemini` separately targets static `google/gemini-2.5-flash` and has no free-tier claim.

Successful responses include safe diagnostic headers that can help server-side debugging:

- `X-Request-ID`
- `X-Go-Ai-Model-Alias`
- `X-Go-Ai-Provider`
- `X-Go-Ai-Upstream-Model`
- `X-Go-Ai-Fallback-Used`
- `X-Go-Ai-Duration-Ms`

Next server code can read and forward these to its own logs or error responses, for example:

```ts
const requestId = response.headers.get("X-Request-ID");
const provider = response.headers.get("X-Go-Ai-Provider");
const fallbackUsed = response.headers.get("X-Go-Ai-Fallback-Used");
const durationMs = response.headers.get("X-Go-Ai-Duration-Ms");
```

Go-Ai also refreshes its in-memory provider model catalog on startup and then hourly by default. The runtime Gemini selection is local to each process, resets/reselects on restart, and retains the last selected direct primary if a later Gemini catalog refresh fails. No Redis is required for this MVP.

The protected model catalog endpoint returns local aliases and candidates together with discovered provider models, refresh diagnostics, and runtime Gemini selection metadata. It is not an upstream OpenAI model-list pass-through:

```sh
curl https://go-ai-i8r-lg.fly.dev/v1/models \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>"
```

## Observability

Go-Ai writes structured safe logs to stdout/stderr. On Fly.io, inspect them with:

```sh
fly logs -a go-ai-i8r-lg
```

The chat log line includes safe metadata such as request ID, route, status, duration, selected provider, upstream model, fallback flag, streaming flag, and error type. It does not log prompts, messages, request/response bodies, tool arguments, tool results, opaque tool metadata, `Authorization` headers, provider keys, or `.env` values.

Runtime counters are exposed through the separate protected runtime status endpoint:

```sh
curl https://go-ai-i8r-lg.fly.dev/v1/status \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>"
```

The response contains a safe in-memory snapshot: uptime, request totals, success/error totals, auth failures, fallback and streaming counters, provider counters, status-code counters, and last request time. These metrics are per process and reset on restart; if the app runs on multiple Fly machines, each machine has its own local counters.

## Tool calling: Variant A flow

Go-Ai does not execute tools, store memory, validate provider-specific tool semantics, or know your application's database, APIs, permissions, or business logic. It only proxies OpenAI-compatible tool-calling payloads after replacing the local model alias.

The client or Next server owns tool execution:

1. Next sends a chat request with `tools` and optional `tool_choice` / `parallel_tool_calls`.
2. Go-Ai resolves the local model alias and forwards the JSON payload to the provider.
3. The model may return assistant `tool_calls`.
4. Next validates and executes the requested tools in application code.
5. Next sends a follow-up chat request that includes each tool result as a message with `role: "tool"` and the matching `tool_call_id`.

Keep the entire assistant tool-call message in history, not just `id`, function name, and arguments. Unknown nested fields are opaque provider metadata and must be sent back unchanged with the matching `role: "tool"` result. For example, a provider may include `extra_content` to continue a tool/function loop; do not assume that every provider needs or emits that field. Direct Gemini 3 function/tool flows can carry metadata that must round-trip, so do not rebuild those messages from a narrow TypeScript type.

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
  "model": "default",
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

Go-Ai does not parse or modify SSE chunks. It resolves the local model alias, forwards the request upstream, then proxies the upstream status, headers, and body back to the caller. Fallback can happen only if the upstream returns a retryable status before Go-Ai starts proxying the response body; once streaming body chunks are being sent to the client, Go-Ai cannot transparently switch to another stream. Streaming tool calls may arrive split across multiple SSE chunks, so your Next app or browser UI must assemble partial deltas losslessly before executing or displaying structured tool-call data. Retain every completed tool-call field, including unknown opaque metadata, in the next-turn history. Tool execution still stays in the client/Next application; Go-Ai only passes payloads through.

Quick deployed smoke test with curl:

```sh
curl -N https://go-ai-i8r-lg.fly.dev/v1/chat/completions \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"default",
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
      model: "default",
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
    // Tool-call deltas may be partial: accumulate every field, including
    // unknown metadata, before using or storing the completed tool call.
    console.log(data);
  }
}
```

For a typing effect in the UI, prefer real provider-to-Next-to-browser streaming for long responses. A client-side typing simulation after receiving the full response is simpler, but it does not reduce time to first token.

## Voice input

Vercel AI SDK is not required for voice input itself. For Groq STT through Go-Ai, the browser records locally and calls the Next app's own upload route; that server-side route streams multipart data to `POST /v1/audio/transcriptions` with the Go-Ai bearer secret. The browser must never call Go-Ai directly.

Every microphone UI using this path must enforce the exact frontend capture contract:

1. Maximum capture is **5:00**.
2. Display a warning at **4:30**.
3. At **5:00**, forcibly stop `MediaRecorder`, finalize the blob, and immediately upload it.
4. Clear timers and stop media tracks on manual stop, cancellation, error, and unmount to avoid duplicate uploads.

Use elapsed wall-clock time and a hard 300,000 ms stop timer, not only recorder chunk events. This is an intentional frontend UX limit, not trusted server validation: a client can bypass it. The Go-Ai STT route instead requires `Content-Length` and rejects requests over `GROQ_STT_MAX_REQUEST_BYTES` (25,000,000 bytes by default). That byte limit protects the server but cannot determine or guarantee audio duration because codecs and bitrates differ. Do not buffer an entire upload to construct a missing length; Go-Ai returns `411 Length Required` when it is absent.

Other voice-input approaches include:

- browser Web Speech API;
- browser `MediaRecorder` plus Go-Ai's Groq STT proxy or another speech-to-text provider;
- an external STT service.

After transcription, send the resulting text to Go-Ai as a normal chat message. Vercel AI SDK may still help with chat state or streaming UI, but it is optional for voice capture/transcription.

## Voice output (TTS)

TTS is an application UX decision, not automatic Go-Ai behavior. Keep the Go-Ai bearer secret in a Next server route that proxies `POST /v1/audio/speech`; browser code must not call Go-Ai directly. Stream the binary response rather than parsing it as JSON, storing it, or fully buffering it.

- In a `ru` interface locale, the first version must hide or disable the speech control and must not call the TTS endpoint.
- In an `en` interface locale, request TTS only after an app-owned reliable language guard verifies that the final assistant text is English. Do not use the UI locale as the guard: the model can return Russian in an English UI.
- When the final text is non-English or its language is indeterminate, leave it as text and skip TTS. Do not translate it automatically to enable speech, and do not promise Russian TTS support.
- This does not limit STT: it remains independent and can be used in `ru` and `en` under the voice-input contract above, without unverified claims about transcription quality.
