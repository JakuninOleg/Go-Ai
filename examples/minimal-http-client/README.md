# Minimal HTTP client example

This is the smallest useful integration path: call Go-Ai from trusted server-side code over HTTP.

Do not run this code in a browser. `GO_AI_SHARED_SECRET` is a backend secret and must stay on your server, worker, CLI, or internal service.

## Environment

```dotenv
GO_AI_BASE_URL=http://localhost:8080
GO_AI_SHARED_SECRET=replace-with-a-random-secret
```

## Health check

```sh
curl "$GO_AI_BASE_URL/health"
```

## Chat request with curl

```sh
curl "$GO_AI_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $GO_AI_SHARED_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      { "role": "user", "content": "Say hello in one sentence." }
    ]
  }'
```

If `model` is omitted, Go-Ai uses the default local alias.

## TypeScript helper

[`chat.ts`](chat.ts) exports one server-side function:

```ts
const result = await askGoAi([
  { role: "user", content: "Say hello in one sentence." },
]);
```

Use the same HTTP shape from any backend language: Rust, Ruby, Python, Go, Next.js route handlers, background jobs, or internal services.
