# Design principles

Go-Ai is a gateway, not an agent framework.

It exists to give server-side applications a small, OpenAI-compatible provider boundary. It should make provider access boring: one protected HTTP API, local model aliases, predictable routing, fallback where it is safe, and streaming pass-through without dragging an orchestration framework into every app.

## Boundary

Applications own product behavior:

- tool execution;
- memory and conversation persistence;
- RBAC, tenant checks, and user permissions;
- business context and data access;
- prompt policy and product-specific workflow.

Go-Ai owns the provider boundary:

- bearer authentication for gateway access;
- local model aliases;
- provider routing;
- retryable fallback between configured candidates;
- provider model discovery and refresh;
- HTTP/SSE streaming pass-through;
- safe diagnostics that do not expose prompts or secrets.

This split keeps domain decisions close to the app that has the user, database, permissions, and audit context.

Agent guidance in [Agent integration guide](agent-integration.md) and [`llms.txt`](../llms.txt) exists to keep generated integrations aligned with this boundary.

## Privacy and logging

Go-Ai must not log prompts, messages, request bodies, response bodies, tool arguments, authorization headers, provider keys, shared secrets, or `.env` values.

Logs and diagnostic headers are for operations: request IDs, status codes, latency, selected provider, resolved upstream model, fallback usage, and broad error categories. They should help debug routing and provider availability without turning the gateway into a prompt recorder.

## Tool calling

Tool calling is pass-through compatibility, not execution.

Go-Ai forwards `tools`, `tool_choice`, assistant `tool_calls`, and follow-up `role: "tool"` messages as OpenAI-compatible JSON. It does not validate business permissions, call APIs, mutate application state, or decide whether a tool is allowed for a user.

The application loop is responsible for:

1. sending tool schemas to the model;
2. receiving tool calls;
3. validating the current user and request context;
4. executing the tool;
5. sending tool results back through Go-Ai for the final model response.

## Model aliases are the app contract

Client apps should depend on local aliases such as `default` or `gemini-flash`, not raw provider slugs. Provider names change, availability changes, and fallback candidates can evolve. The alias is the public contract between apps and Go-Ai; provider-specific model names are an implementation detail behind that contract.

Go-Ai refreshes provider model catalogs automatically inside the running process so operators can inspect availability without constantly checking provider lists by hand. That autonomy does not make discovery an automatic router: the static alias registry remains the stable baseline, and Go-Ai does not blindly move apps to the newest discovered provider model.

Changing alias behavior should be treated as a compatibility change and covered by tests.

For the practical extension workflow, see [Adding models and providers](adding-models.md).

## Provider scope in v0.1

v0.1 intentionally supports a small provider set: Gemini as the default provider and OpenRouter as the fallback/aggregator route. That keeps the first release readable, testable, and honest about what it has actually been built to support.

Provider expansion is welcome when it preserves the gateway boundary: provider-specific code belongs in `internal/providers`, aliases stay in the local registry, and successful upstream responses remain proxied without handler-level special cases.

## Non-goals

Go-Ai intentionally does not include:

- prompt template management;
- chains or agent planners;
- vector databases or RAG pipelines;
- tool executors;
- dashboards;
- billing systems;
- enterprise multi-tenancy;
- application memory;
- product-specific moderation or policy engines.

Those features belong either in the application or in a broader LLM platform.

## Why smaller than LiteLLM or LangChain

LiteLLM, LangChain, and similar projects are useful when you need broad provider coverage, plugin ecosystems, tracing integrations, chains, agents, or larger orchestration surfaces.

Go-Ai is deliberately smaller. It is meant to be easy to read, cheap to run, and safe to place between applications or client backends and a small set of providers. A Next.js app is one common client, not the only one. The tradeoff is clear: fewer features, fewer moving parts, and a sharper boundary. If an app needs a full orchestration framework, it should use one. If it needs a thin provider gateway with local aliases and streaming pass-through, Go-Ai should stay focused on that job.
