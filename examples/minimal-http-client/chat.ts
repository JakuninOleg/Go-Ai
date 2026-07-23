type ChatMessage = {
  role: "system" | "user" | "assistant" | "tool";
  content: string | null;
  name?: string;
  tool_call_id?: string;
};

type AskGoAiOptions = {
  model?: string;
  baseUrl?: string;
  sharedSecret?: string;
};

export async function askGoAi(
  messages: ChatMessage[],
  options: AskGoAiOptions = {},
) {
  const baseUrl = options.baseUrl ?? process.env.GO_AI_BASE_URL ?? "http://localhost:8080";
  const sharedSecret = options.sharedSecret ?? process.env.GO_AI_SHARED_SECRET;

  if (!sharedSecret) {
    throw new Error("GO_AI_SHARED_SECRET is required");
  }

  const response = await fetch(`${baseUrl}/v1/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${sharedSecret}`,
    },
    body: JSON.stringify({
      messages,
      ...(options.model ? { model: options.model } : {}),
    }),
  });

  const body = await response.json();

  if (!response.ok) {
    throw new Error(`Go-Ai request failed with ${response.status}: ${JSON.stringify(body)}`);
  }

  return body;
}
