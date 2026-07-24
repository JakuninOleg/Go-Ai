# Start here: integrate Go-Ai into another project

Use this handoff when you want a coding agent to connect another application to Go-Ai without giving it this whole repository.

## Step 1: Copy these files into the target app

Create `docs/go-ai/` in the target app and copy:

- `llms.txt` -> `docs/go-ai/llms.txt`
- `docs/agent-integration.md` -> `docs/go-ai/agent-integration.md`
- `docs/next-client.md` -> `docs/go-ai/next-client.md` if the target app is Next.js

Copy prompt files into `docs/go-ai/prompts/` as needed:

- `docs/agent-prompts/connect-go-ai.md`
- `docs/agent-prompts/migrate-from-vercel-ai-sdk.md` if migrating from Vercel AI SDK or another provider SDK
- `docs/agent-prompts/add-streaming.md` if streaming is needed
- `docs/agent-prompts/add-tool-calling.md` if tools, memory, or actions are needed

## Step 2: Add server-side env vars in the target app

Add these to the target app's server-side environment:

```dotenv
GO_AI_BASE_URL=https://your-go-ai.example.com
GO_AI_SHARED_SECRET=replace-with-your-gateway-secret
```

Do not expose `GO_AI_SHARED_SECRET` in browser code, public env vars, logs, screenshots, or committed files.

## Step 3: Give the agent one prompt

Choose the smallest relevant prompt file from `docs/go-ai/prompts/`, then give the agent this:

```text
We are in the <APP_NAME> project. Migrate AI integration to Go-Ai.

Read:
- docs/go-ai/llms.txt
- docs/go-ai/agent-integration.md
- docs/go-ai/prompts/<CHOSEN_PROMPT>.md
- docs/go-ai/next-client.md (if this is a Next.js app)

Goals:
- remove direct provider/model calls from this app;
- send LLM requests through Go-Ai;
- use GO_AI_BASE_URL and GO_AI_SHARED_SECRET only server-side;
- do not expose secrets in browser code or public env vars;
- preserve the current UI/API contract where possible;
- if streaming exists, preserve it via stream:true;
- if tools/memory/actions exist, execute them in this app; Go-Ai only proxies tool_calls;
- run the project checks after changes.

First audit the current AI integration and propose a migration plan before editing files.
```

If you want the agent to decide the implementation route, add:

```text
Decide the smallest safe integration path after the audit.
```

After implementation, run the app locally and verify the existing chat UI, API route, worker, or service path still works.
