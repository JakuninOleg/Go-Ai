type ChatMessage = {
  role: "system" | "user" | "assistant" | "tool";
  content?: string | null;
  name?: string;
  tool_call_id?: string;
  tool_calls?: unknown[];
};

export type ChatCompletionRequest = {
  model?: string;
  messages: ChatMessage[];
  stream?: boolean;
  tools?: unknown[];
  tool_choice?: unknown;
  temperature?: number;
};

export function getGoAiConfig() {
  const baseUrl = process.env.GO_AI_BASE_URL;
  const sharedSecret = process.env.GO_AI_SHARED_SECRET;

  if (!baseUrl) {
    throw new Error("GO_AI_BASE_URL is not configured");
  }

  if (!sharedSecret) {
    throw new Error("GO_AI_SHARED_SECRET is not configured");
  }

  return {
    baseUrl: baseUrl.replace(/\/$/, ""),
    sharedSecret,
  };
}

export async function fetchGoAiChat(body: ChatCompletionRequest) {
  const { baseUrl, sharedSecret } = getGoAiConfig();

  return fetch(`${baseUrl}/v1/chat/completions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${sharedSecret}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
}

export function selectedGoAiHeaders(upstream: Response) {
  const headers = new Headers();

  for (const name of [
    "content-type",
    "x-request-id",
    "x-go-ai-model-alias",
    "x-go-ai-provider",
    "x-go-ai-upstream-model",
    "x-go-ai-fallback-used",
    "x-go-ai-duration-ms",
  ]) {
    const value = upstream.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }

  return headers;
}
