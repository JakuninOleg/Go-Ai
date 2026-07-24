# Prompt: migrate from Vercel AI SDK to Go-Ai

Copy this prompt into a coding agent when migrating a Next.js or similar app from Vercel AI SDK/provider SDK calls to Go-Ai.

```text
You are working in the target application repository.

Goal: migrate the app's LLM provider boundary from Vercel AI SDK/provider SDK calls to Go-Ai, while preserving the existing user-facing behavior.

Context:
- Go-Ai base URL: <GO_AI_BASE_URL>
- Main chat/API route: <ROUTE_PATH>
- Current UI entry points/hooks: <UI_PATHS_OR_UNKNOWN>
- Current behavior: <STREAMING_OR_NON_STREAMING_OR_UNKNOWN>
- Tool/function calling used today: <YES_NO_UNKNOWN>

Go-Ai contract:
- Browser code must call the app's own backend route, not Go-Ai directly.
- Backend code calls POST ${GO_AI_BASE_URL}/v1/chat/completions.
- Backend code sends Authorization: Bearer ${GO_AI_SHARED_SECRET}.
- GO_AI_SHARED_SECRET is server-side only. Do not use NEXT_PUBLIC_, PUBLIC_, VITE_, or any client-exposed env prefix for it.
- Use OpenAI-compatible messages, model aliases, stream, tools, and tool_choice fields.
- If the existing UI streams, keep streaming by sending stream: true to Go-Ai and proxying the response body. Do not call response.json() on a stream.
- If the existing UI does not stream, parse Go-Ai's JSON response and preserve the existing route response shape.
- Tool execution stays in the app. Go-Ai only proxies tool-calling payloads.
- Do not log prompts, messages, request bodies, response bodies, tool arguments, bearer tokens, provider keys, or env values.

Tasks:
1. Inspect package files and source for ai, @ai-sdk/*, streamText, generateText, useChat, OpenAI, GoogleGenerativeAI, and provider-specific clients.
2. Map the current request/response contract between UI and backend route.
3. Replace server-side provider SDK calls with fetch calls to Go-Ai.
4. Preserve auth/session checks, rate limiting, validation, and existing UI behavior.
5. Keep streaming if the current UI streams. Otherwise keep non-streaming behavior.
6. Preserve or adapt tool-calling loops so tools execute only in the app with permission checks and max iterations.
7. Remove unused provider keys from the target app's env docs when they are no longer needed there.
8. Add GO_AI_BASE_URL and GO_AI_SHARED_SECRET placeholders to env examples/docs.
9. Update tests or add coverage for the route/helper changed.
10. Run the target app's relevant validation commands.

Constraints:
- Do not rewrite unrelated UI.
- Do not move app tools, database access, permissions, memory, or business logic into Go-Ai.
- Do not expose secrets in client bundles.
- Do not invent real keys or URLs; use placeholders.

Deliverables:
- Changed files.
- Migration summary: old provider path -> new Go-Ai path.
- Streaming decision and why.
- Tool-calling decision, if applicable.
- Env/doc updates.
- Validation commands and results.
```
