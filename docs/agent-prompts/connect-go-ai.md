# Prompt: connect an app to Go-Ai

Copy this prompt into a coding agent when you want a minimal non-streaming Go-Ai integration.

```text
You are working in the target application repository.

Goal: connect this app to Go-Ai through a server-side HTTP integration.

Context:
- Go-Ai base URL: <GO_AI_BASE_URL>
- Target app route/helper to update or create: <PATH_OR_DESCRIPTION>
- Desired model alias, if any: <MODEL_ALIAS_OR_DEFAULT>
- Existing UI/API contract to preserve: <DESCRIBE_CONTRACT>

Go-Ai contract:
- Call Go-Ai from server-side/backend code only.
- Use POST ${GO_AI_BASE_URL}/v1/chat/completions.
- Add headers: Content-Type: application/json and Authorization: Bearer ${GO_AI_SHARED_SECRET}.
- Send an OpenAI-compatible body with messages and optional model.
- For this task, implement non-streaming JSON first unless the existing route already streams.
- Do not expose GO_AI_SHARED_SECRET in browser code, public env vars, logs, or client bundles.
- Do not log prompts, messages, tool args, request bodies, response bodies, or secrets.

Tasks:
1. Inspect the existing app structure, routes, env handling, tests, and docs.
2. Add or update server-side env usage for GO_AI_BASE_URL and GO_AI_SHARED_SECRET.
3. Create or update a small backend-only helper for Go-Ai chat completions.
4. Wire the helper into the target route while preserving the current UI/API response shape.
5. Check response.ok and return a safe app-level error when Go-Ai fails.
6. Update .env.example or setup docs with placeholders only.
7. Add or update tests for the changed route/helper where practical.
8. Run the relevant validation commands for this repository.

Deliverables:
- List changed files.
- Explain how the app calls Go-Ai.
- Confirm that GO_AI_SHARED_SECRET remains server-side only.
- Report validation commands and results.
```
