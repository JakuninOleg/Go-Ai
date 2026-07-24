# Adding models and providers

Go-Ai keeps model selection behind local aliases. Client applications send stable names such as `default`, `gemini-flash`, or another alias you define. The gateway resolves that alias to one or more provider-specific model slugs before forwarding the request upstream.

This is intentional: provider model names change, free tiers appear and disappear, and fallback candidates may need to move without forcing every client app to change its request payloads.

## Why v0.1 starts with Gemini and OpenRouter

v0.1 keeps provider coverage narrow on purpose.

- **Gemini is the default provider** because it is the primary provider this gateway was built around for personal server-side apps, and it exposes an OpenAI-compatible endpoint that fits Go-Ai's proxy boundary.
- **OpenRouter is the fallback and aggregator option** because it gives access to many models through one OpenAI-compatible API and can provide free or low-cost fallback candidates.
- **The first release should stay small and testable.** Go-Ai is not trying to be a universal provider marketplace. It is a focused gateway with local aliases, predictable routing, streaming pass-through, and safe diagnostics.

Additional providers can be added through the provider interface when there is a real need and enough tests to keep the gateway behavior predictable.

## Add a model for an existing provider

For Gemini or OpenRouter, most model additions start in [`internal/models/registry.go`](../internal/models/registry.go).

1. Choose the local alias clients should use.
2. Verify the provider's real model slug against the provider's current documentation or API before relying on it in production.
3. Add an entry to `AliasRegistry` with one or more `ModelConfig` candidates.
4. Add or update the compatibility entry in `Registry` if code still needs direct alias-to-primary-model lookup.
5. Add or update tests in `internal/models` and `internal/services` when alias resolution, fallback order, or error behavior changes.
6. Run:

   ```sh
   gofmt -w <changed-go-files>
   go test ./...
   ```

Example shape:

```go
"fast-chat": {
	Candidates: []ModelConfig{
		{
			Name:     "provider/model-slug",
			Provider: ProviderOpenRouter,
		},
	},
},
```

Use `ProviderGemini` for Gemini model slugs and `ProviderOpenRouter` for OpenRouter model slugs.

## Discovery does not replace aliases

Go-Ai refreshes provider model catalogs in memory on startup and at the configured `MODEL_REFRESH_INTERVAL`. This helps operators inspect what providers currently report through `GET /v1/models` and reduces constant manual checking for model availability.

Discovery is diagnostic information, not the public app contract. For v0.1, adding or changing public model names still happens in the static alias registry. Do not route an app to a newly discovered provider slug just because it appears in the catalog; first decide whether it should become an alias candidate, verify the behavior you need, and cover the alias behavior with tests.

## Add or change fallback candidates

Fallback is controlled by the ordered `Candidates` list for an alias in `AliasRegistry`.

```go
DefaultModelAlias: {
	Candidates: []ModelConfig{
		{
			Name:     "primary-model-slug",
			Provider: ProviderGemini,
		},
		{
			Name:     "fallback-model-slug",
			Provider: ProviderOpenRouter,
		},
	},
},
```

Put the preferred candidate first. Go-Ai only tries the next candidate when the previous attempt fails in a retryable way, such as a network error before a response is received or an upstream `429`, `500`, `502`, `503`, or `504`.

Do not use fallback to hide invalid requests, auth failures, missing API keys, or unsupported model features. Those should fail clearly.

## Add a new provider

Provider expansion should keep Go-Ai's gateway boundary intact: resolve aliases, forward OpenAI-compatible payloads, proxy responses, and avoid provider-specific logic in HTTP handlers.

Typical steps:

1. Add provider constants or model aliases in `internal/models/registry.go`.
2. Implement `providers.Provider` in `internal/providers`:

   ```go
   type Provider interface {
    Chat(ctx context.Context, body []byte) (*http.Response, error)
   }
   ```

3. Implement `providers.ModelLister` when the provider has a compatible model-listing API:

   ```go
   type ModelLister interface {
    ListModels(ctx context.Context) ([]ModelInfo, error)
   }
   ```

4. Add config fields and environment variables in `internal/config/config.go`.
5. Update `.env.example`, README configuration docs, and deployment docs if new env vars are required.
6. Register the provider in `internal/providers/router.go` so aliases can resolve to it and model discovery can refresh its catalog.
7. Wire the provider in `cmd/api/main.go`.
8. Add tests for provider routing, alias resolution, fallback behavior, missing API keys, and error responses.
9. Run `gofmt -w <changed-go-files>`, `go test ./...`, and `go vet ./...`.

If the provider does not expose an OpenAI-compatible chat API, keep the translation layer inside that provider implementation. Handlers and services should not become a collection of provider-specific branches.

## Capability caveats

OpenAI-compatible JSON does not mean every model supports every feature.

Before promoting a model alias for real use, test the behavior you plan to rely on:

- normal chat completions;
- `stream: true` HTTP/SSE streaming;
- tool-calling payload pass-through if your app sends `tools`, `tool_choice`, assistant `tool_calls`, or `role: "tool"` follow-up messages;
- expected error shape for invalid requests or unsupported features.

Go-Ai passes tool-calling payloads through, but it does not execute tools. Tool execution stays in the calling application or service.

## Future registry options

A file-based or environment-based registry may be useful later for deployments that need to tune aliases without rebuilding the binary.

For v0.1, the registry stays in code because it is simple, reviewable, and easy to cover with tests.
