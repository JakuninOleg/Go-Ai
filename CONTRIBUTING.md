# Contributing

Thanks for taking the time to improve Go-Ai.

## Local checks

Before opening a pull request, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

If you add or change model alias behavior, update the related tests in `internal/models` and `internal/services`.

## Security and privacy

- Never commit `.env`, provider API keys, shared secrets, tokens, or real user data.
- Do not log request bodies, provider keys, `GO_AI_SHARED_SECRET`, or `.env` values.
- Keep example configuration files limited to placeholders and public provider base URLs.

## Architecture boundaries

Go-Ai is a small OpenAI-compatible gateway and model router. Keep provider access, alias resolution, request validation, and HTTP proxy behavior inside this project.

Tool execution intentionally belongs to client applications, such as Next apps, because those apps own business logic, databases, permissions, and user context. Please do not add business-specific tool executors to Go-Ai.

## Adding providers or models

- Prefer local model aliases over exposing provider model slugs to clients.
- Add provider-specific code under `internal/providers` and wire it through the provider router rather than special-casing handlers.
- Keep successful upstream OpenAI-compatible request/response proxying intact.
- Verify provider model slugs against official provider documentation before relying on them in production.
- Add tests for alias resolution, fallback behavior, and error responses when behavior changes.
