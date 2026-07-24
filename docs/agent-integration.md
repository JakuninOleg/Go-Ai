# Agent integration guide

This guide is for developers and coding agents connecting an application to Go-Ai. It is generic first: any backend, route handler, worker, or internal service can call Go-Ai over HTTP. Next.js examples are linked where useful, but Go-Ai is not a Next-only project.

Go-Ai is the provider gateway. The target application remains responsible for product behavior, user context, permissions, persistence, UI contracts, and tool execution.

## Using this guide in another project

For a short handoff to a coding agent, start with [Start here: integrate Go-Ai into another project](agent-prompts/start-here.md).

Copy `llms.txt`, this guide, and the relevant prompt files into the target app under `docs/go-ai/`. If the target app is Next.js, copy `docs/next-client.md` too. The target app should then add server-side `GO_AI_BASE_URL` and `GO_AI_SHARED_SECRET` values and ask the agent to audit the existing AI integration before editing files.

## What agents should know before editing an app

Before changing code, inspect the target app and answer these questions:

1. Where does the app currently call an LLM provider or AI SDK?
2. Is the call server-side only, or is provider access exposed to browser code?
3. Does the current UI expect a full JSON response, a text stream, SSE, or another stream format?
4. Does the route use tools/function calling? If yes, where are permissions and side effects handled?
5. Which tests, docs, and env examples describe the existing AI integration?

Do not rewrite unrelated UI or application architecture. Replace the provider boundary while preserving the app's public behavior unless the task explicitly asks for a behavior change.

## Go-Ai HTTP contract

Use the OpenAI-compatible chat completions endpoint:

```http
POST /v1/chat/completions
Authorization: Bearer <GO_AI_SHARED_SECRET>
Content-Type: application/json
```

Request body:

```json
{
  "model": "default",
  "messages": [
    { "role": "user", "content": "Say hello in one sentence." }
  ]
}
```

Notes:

- `model` is optional. If omitted, Go-Ai uses its default local alias.
- Prefer Go-Ai local aliases over provider model slugs in client applications.
- Send OpenAI-compatible fields such as `messages`, `stream`, `temperature`, `tools`, and `tool_choice` as needed.
- Go-Ai resolves the local alias, forwards the request upstream, and proxies the upstream response.
- Check `response.ok` before returning data to the UI.

Minimal server-side TypeScript call:

```ts
const baseUrl = process.env.GO_AI_BASE_URL ?? "http://localhost:8080";
const sharedSecret = process.env.GO_AI_SHARED_SECRET;

if (!sharedSecret) {
  throw new Error("GO_AI_SHARED_SECRET is required");
}

const response = await fetch(`${baseUrl}/v1/chat/completions`, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: `Bearer ${sharedSecret}`,
  },
  body: JSON.stringify({
    model: "default",
    messages: [{ role: "user", content: "Say hello in one sentence." }],
  }),
});

if (!response.ok) {
  throw new Error(`Go-Ai request failed with status ${response.status}`);
}

const data = await response.json();
```

For complete examples, see:

- [Minimal HTTP client](../examples/minimal-http-client)
- [Next.js route handler](../examples/next-route-handler)
- [Next.js client integration guide](next-client.md)

## Required env vars in the target app

The target application should keep these values server-side:

```dotenv
GO_AI_BASE_URL=http://localhost:8080
GO_AI_SHARED_SECRET=replace-with-your-gateway-secret
```

Rules:

- Do not prefix the shared secret with public/client env prefixes such as `NEXT_PUBLIC_`, `PUBLIC_`, or `VITE_`.
- Do not expose the shared secret in browser code or static assets.
- Do not commit real `.env` files.
- Keep provider API keys in the Go-Ai deployment, not in the target app, unless that app still calls a provider for a separate feature.

## Minimal integration checklist

Use this checklist for a non-streaming integration:

1. Add server-side env vars for `GO_AI_BASE_URL` and `GO_AI_SHARED_SECRET`.
2. Create or update a backend-only helper that calls `POST /v1/chat/completions`.
3. Send OpenAI-compatible `messages` and an optional local `model` alias.
4. Add bearer auth and `Content-Type: application/json`.
5. Check `response.ok` and return a safe error shape from the app route.
6. Parse non-streaming responses with `response.json()`.
7. Preserve the existing route response shape expected by the UI.
8. Update env examples, setup docs, and tests.
9. Remove unused provider SDK keys from the target app if they are no longer needed.

## Migrating from Vercel AI SDK

The goal is to replace direct provider SDK calls with a server-side HTTP call to Go-Ai while preserving the app's UI behavior.

Before editing:

- Find routes using `ai`, `@ai-sdk/*`, `streamText`, `generateText`, `OpenAI`, `GoogleGenerativeAI`, or provider-specific clients.
- Find UI hooks such as `useChat` and check which response format they expect.
- Identify whether the current route streams.
- Identify tool/function definitions and where tool execution happens.
- Read existing tests and route docs before changing the contract.

Migration steps:

1. Keep the browser talking to the app's own API route, not directly to Go-Ai.
2. Replace provider SDK calls inside the server-side route with `fetch` to Go-Ai.
3. Preserve request parsing, auth/session checks, rate limits, and UI response shape.
4. If the current UI streams, keep streaming and proxy `response.body` from Go-Ai through the app route.
5. If the current UI does not stream, use a normal JSON response and parse `response.json()`.
6. Move provider API keys out of the target app when they are no longer used there.
7. Keep `GO_AI_SHARED_SECRET` server-side only.
8. Update docs, env examples, and tests to describe Go-Ai instead of the provider SDK boundary.

Do not remove Vercel AI SDK UI hooks just because the backend provider changed. If `useChat` or another UI contract is working, adapt the server route to keep that contract unless the task asks for a UI migration too.

## Streaming checklist

Use streaming when the target UI already streams or when the task explicitly asks for streaming.

1. Send `stream: true` in the Go-Ai request body.
2. Check that `response.ok` before returning the stream.
3. Return or pipe `response.body` from the server route.
4. Forward useful response headers such as `Content-Type` when appropriate.
5. Do not call `response.json()` on a streaming response.
6. Do not buffer the whole stream unless the app contract requires buffering.
7. Decide how the app handles upstream errors before and after streaming starts.
8. Test with a real streaming request and verify the UI receives incremental output.

Go-Ai proxies upstream SSE chunks. It does not parse, merge, or rewrite active streams.

## Tool-calling checklist

Go-Ai supports tool-calling payload compatibility, not tool execution.

When adding tool calling to an app:

1. Define tool schemas in the target app.
2. Send `tools` and optional `tool_choice` to Go-Ai.
3. Receive assistant `tool_calls` from the proxied model response.
4. Validate the current user, tenant, permissions, and input arguments before executing a tool.
5. Execute tools in the target app or its trusted backend services.
6. Append each tool result as an OpenAI-compatible `role: "tool"` message.
7. Send the follow-up messages through Go-Ai to get the final assistant response.
8. Use a maximum iteration limit to avoid infinite tool loops.
9. Log only safe metadata; never log tool arguments or secrets.

Do not add business-specific tools, database access, or app permissions to Go-Ai.

## Security rules

- Call Go-Ai from trusted server-side code only.
- Never expose `GO_AI_SHARED_SECRET` to the browser.
- Never commit real `.env` values.
- Never log prompts, messages, request bodies, response bodies, tool arguments, bearer tokens, provider API keys, shared secrets, or `.env` values.
- Preserve app-level authentication, authorization, rate limiting, and audit behavior.
- Treat model output as untrusted input.
- Validate tool arguments against the current user and request context before executing anything.

## Validation checklist

For target app integrations, run the narrowest useful checks the app provides:

1. Typecheck or build.
2. Unit tests for the changed helper or route.
3. Existing route/API tests.
4. Lint if the project uses it.
5. A local manual request against the app route.
6. A streaming manual check if streaming changed.
7. A tool-call path check if tools changed.

For changes in this Go-Ai repository, run:

```sh
go test ./...
go vet ./...
git diff --check
```
