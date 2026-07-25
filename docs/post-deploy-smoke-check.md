# Post-deploy smoke check

Use [`scripts/post-deploy-smoke-check.sh`](../scripts/post-deploy-smoke-check.sh) after a deployment to verify the public health endpoint and a real non-streaming chat request. It can also check that a streaming request receives a successful HTTP status.

The script calls external providers and can consume quota or incur usage. It deliberately prints only its check name and HTTP status: never credentials, request/response headers, or response bodies.

## Required environment

Inject these values through your shell, secret manager, or CI secret store; do not put them in a committed file:

```sh
export GO_AI_BASE_URL=https://your-go-ai-host
export GO_AI_SHARED_SECRET=...
```

Run the mandatory health and non-streaming checks:

```sh
sh scripts/post-deploy-smoke-check.sh
```

To also issue one SSE request, set `GO_AI_SMOKE_STREAM=1`:

```sh
GO_AI_SMOKE_STREAM=1 sh scripts/post-deploy-smoke-check.sh
```

`GO_AI_SMOKE_TIMEOUT_SECONDS` defaults to `30` and may be overridden for a slower deployment path.

This is intentionally a manual or separately credentialed deployment check, not a CI unit test. The regular test suite must remain deterministic and must not require provider keys or live network access.
