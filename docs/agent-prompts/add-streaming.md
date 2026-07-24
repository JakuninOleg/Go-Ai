# Prompt: add streaming through Go-Ai

Copy this prompt into a coding agent when an app already uses Go-Ai or needs streaming added through Go-Ai.

```text
You are working in the target application repository.

Goal: implement streaming chat responses through Go-Ai while preserving the app's route/UI contract.

Context:
- Go-Ai base URL: <GO_AI_BASE_URL>
- Route/helper to update: <PATH_OR_DESCRIPTION>
- UI stream format expected today: <SSE_READABLE_STREAM_TEXT_OR_UNKNOWN>
- Existing non-streaming helper, if any: <PATH_OR_NONE>

Go-Ai streaming contract:
- Call Go-Ai from server-side/backend code only.
- Use POST ${GO_AI_BASE_URL}/v1/chat/completions.
- Send Authorization: Bearer ${GO_AI_SHARED_SECRET}.
- Include stream: true in the JSON body.
- Check response.ok before returning the stream.
- Read, return, or pipe response.body as a stream.
- Do not call response.json() on a streaming response.
- Do not expose GO_AI_SHARED_SECRET in browser code or public env vars.
- Do not log prompts, messages, request bodies, response bodies, chunks, or secrets.

Tasks:
1. Inspect the current route and UI to determine the exact stream format expected.
2. Update the server-side Go-Ai request to include stream: true.
3. Return or transform the Go-Ai response stream only as needed to preserve the existing UI contract.
4. Forward appropriate headers such as Content-Type when safe and useful.
5. Implement safe error handling before streaming starts.
6. Avoid buffering the full response unless the existing UI contract requires it.
7. Update tests/docs/env examples as needed.
8. Run the relevant validation commands and perform a local manual streaming check if possible.

Deliverables:
- Changed files.
- Explanation of how the stream flows from app route to Go-Ai and back to UI.
- Confirmation that streaming code does not use response.json().
- Validation commands and results.
```
