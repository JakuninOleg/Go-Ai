# Prompt: add app-owned tool calling with Go-Ai

Copy this prompt into a coding agent when adding tool/function calling while keeping tool execution in the target app.

```text
You are working in the target application repository.

Goal: implement an app-owned tool-calling loop that uses Go-Ai only as the OpenAI-compatible model gateway.

Context:
- Go-Ai base URL: <GO_AI_BASE_URL>
- Route/helper to update: <PATH_OR_DESCRIPTION>
- Tools to expose: <TOOL_NAMES_AND_PURPOSES>
- User/session/tenant permission source: <AUTH_CONTEXT>
- Maximum tool iterations: <NUMBER, e.g. 3>

Go-Ai tool-calling contract:
- Go-Ai proxies OpenAI-compatible tools, tool_choice, assistant tool_calls, and role: "tool" messages.
- Go-Ai does not execute tools.
- The target app validates permissions, validates arguments, executes tools, handles side effects, and records any required audit events.
- Call Go-Ai from server-side/backend code only.
- Send Authorization: Bearer ${GO_AI_SHARED_SECRET} from server-side code only.
- Do not log prompts, messages, tool arguments, request bodies, response bodies, bearer tokens, provider keys, or env values.

Tasks:
1. Inspect the target app's auth/session model, existing tool-like actions, tests, and route contract.
2. Define OpenAI-compatible tool schemas in the app layer.
3. Send tools and optional tool_choice to Go-Ai with the chat messages.
4. Detect assistant tool_calls in the Go-Ai response.
5. For each tool call, validate the current user/session/tenant and parse arguments with a strict schema.
6. Execute only allowed app-owned tools.
7. Append tool results as OpenAI-compatible role: "tool" messages with the matching tool_call_id.
8. Send follow-up messages through Go-Ai for the final assistant answer.
9. Enforce the configured maximum iteration count to avoid infinite loops.
10. Add tests for allowed tools, rejected permissions/arguments, and max-iteration behavior.
11. Run the relevant validation commands.

Constraints:
- Do not move tools, database access, permissions, or business logic into Go-Ai.
- Do not execute tool calls without permission and argument validation.
- Do not invent tools beyond the requested list.
- Do not expose GO_AI_SHARED_SECRET to browser code.

Deliverables:
- Changed files.
- Tool loop summary with max iterations.
- Security checks added or preserved.
- Tests/validation commands and results.
```
