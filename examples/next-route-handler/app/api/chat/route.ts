import { fetchGoAiChat, selectedGoAiHeaders } from "../../../lib/go-ai";

export const runtime = "nodejs";

type BrowserChatRequest = {
  message?: string;
  stream?: boolean;
};

export async function POST(request: Request) {
  const input = (await request.json()) as BrowserChatRequest;
  const message = input.message?.trim();

  if (!message) {
    return Response.json({ error: "message is required" }, { status: 400 });
  }

  if (input.stream) {
    return streamChat(message);
  }

  return completeChat(message);
}

async function completeChat(message: string) {
  const upstream = await fetchGoAiChat({
    model: "default",
    messages: [{ role: "user", content: message }],
  });

  const headers = selectedGoAiHeaders(upstream);
  const payload = await upstream.json();

  return Response.json(payload, {
    status: upstream.status,
    headers,
  });
}

async function streamChat(message: string) {
  const upstream = await fetchGoAiChat({
    model: "default",
    messages: [{ role: "user", content: message }],
    stream: true,
  });

  if (!upstream.body) {
    return Response.json(
      { error: "Go-Ai returned an empty stream" },
      { status: 502, headers: selectedGoAiHeaders(upstream) },
    );
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: selectedGoAiHeaders(upstream),
  });
}
