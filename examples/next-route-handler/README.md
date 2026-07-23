# Advanced Next.js route-handler examples

These snippets show one practical way to call Go-Ai from a Next.js app without exposing gateway secrets to the browser.

For the smallest integration, start with [`../minimal-http-client`](../minimal-http-client). Go-Ai itself is a generic HTTP/OpenAI-compatible gateway; this directory is an advanced Next.js example with streaming and a tool-calling skeleton.

They are examples, not a complete app. Copy the files into a Next project and adapt validation, auth, and UI behavior to your product.

## Environment

Keep these values server-side only:

```dotenv
GO_AI_BASE_URL=http://localhost:8080
GO_AI_SHARED_SECRET=replace-with-a-random-secret
```

Do not prefix them with `NEXT_PUBLIC_`, and do not send `GO_AI_SHARED_SECRET` to the browser.

## Files

- [`lib/go-ai.ts`](lib/go-ai.ts) contains a small server-side fetch helper.
- [`app/api/chat/route.ts`](app/api/chat/route.ts) shows non-streaming and streaming route handlers.
- [`lib/tool-loop.ts`](lib/tool-loop.ts) sketches the app-owned tool-calling loop.

## What this demonstrates

- Reading `GO_AI_BASE_URL` and `GO_AI_SHARED_SECRET` on the server.
- Sending a non-streaming OpenAI-compatible chat request.
- Returning an upstream SSE body with `new Response(upstream.body, ...)`.
- Forwarding selected diagnostic headers from Go-Ai.
- Keeping tool execution inside the Next.js app while Go-Ai only proxies model requests.

## Browser flow

The browser should call your Next route, for example `/api/chat`. The Next route calls Go-Ai using the shared secret. This keeps provider keys and the Go-Ai bearer token out of client code.
