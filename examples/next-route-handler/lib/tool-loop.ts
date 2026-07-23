import { fetchGoAiChat } from "./go-ai";

type ToolCall = {
  id: string;
  type: "function";
  function: {
    name: string;
    arguments: string;
  };
};

const tools = [
  {
    type: "function",
    function: {
      name: "get_order_status",
      description: "Look up an order status for the current signed-in user.",
      parameters: {
        type: "object",
        properties: {
          orderId: { type: "string" },
        },
        required: ["orderId"],
      },
    },
  },
];

export async function runToolCallingLoop(userId: string, message: string) {
  const first = await fetchGoAiChat({
    model: "default",
    messages: [{ role: "user", content: message }],
    tools,
  });

  const firstPayload = await first.json();
  const assistantMessage = firstPayload.choices?.[0]?.message;
  const toolCalls = (assistantMessage?.tool_calls ?? []) as ToolCall[];

  if (toolCalls.length === 0) {
    return firstPayload;
  }

  const toolResults = await Promise.all(
    toolCalls.map(async (toolCall) => ({
      role: "tool" as const,
      tool_call_id: toolCall.id,
      content: JSON.stringify(await executeToolForUser(userId, toolCall)),
    })),
  );

  const final = await fetchGoAiChat({
    model: "default",
    messages: [
      { role: "user", content: message },
      assistantMessage,
      ...toolResults,
    ],
    tools,
  });

  return final.json();
}

async function executeToolForUser(userId: string, toolCall: ToolCall) {
  if (toolCall.function.name !== "get_order_status") {
    throw new Error(`Unsupported tool: ${toolCall.function.name}`);
  }

  const args = JSON.parse(toolCall.function.arguments) as { orderId?: string };

  if (!args.orderId) {
    throw new Error("orderId is required");
  }

  // This is where the Next app enforces auth, tenant boundaries, and audit rules.
  // Go-Ai only proxies the model request; it does not execute this function.
  return getOrderStatusForUser(userId, args.orderId);
}

async function getOrderStatusForUser(userId: string, orderId: string) {
  return {
    orderId,
    userId,
    status: "processing",
  };
}
