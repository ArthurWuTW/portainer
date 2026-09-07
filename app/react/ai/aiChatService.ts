import type { ChatMessage, LLMConfig } from './types';

export interface StreamHandlers {
  onToken(delta: string): void;
  onToolStart(name: string, arguments_: string): void;
  onToolEnd(name: string, ok: boolean): void;
  onDone(): void;
  onError(message: string): void;
}

interface SSEEvent {
  event: string;
  data: string;
}

function parseSSEBlock(block: string): SSEEvent | null {
  let event = 'message';
  let data = '';
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim();
    } else if (line.startsWith('data:')) {
      data += line.slice(5).trim();
    }
  }
  if (!data) {
    return null;
  }
  return { event, data };
}

function dispatchSSEEvent(event: SSEEvent, handlers: StreamHandlers) {
  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(event.data) as Record<string, unknown>;
  } catch {
    return;
  }

  switch (event.event) {
    case 'token':
      handlers.onToken(String(payload.delta ?? ''));
      break;
    case 'tool_start':
      handlers.onToolStart(String(payload.name ?? ''), String(payload.arguments ?? ''));
      break;
    case 'tool_end':
      handlers.onToolEnd(String(payload.name ?? ''), Boolean(payload.ok));
      break;
    case 'done':
      handlers.onDone();
      break;
    case 'error':
      handlers.onError(String(payload.message ?? 'Unknown error'));
      break;
    default:
      break;
  }
}

// streamChat POSTs the conversation to the Portainer AI endpoint and consumes
// the server-sent event stream, invoking the provided handlers.
export async function streamChat(
  messages: ChatMessage[],
  llmConfig: LLMConfig,
  handlers: StreamHandlers,
  signal?: AbortSignal
): Promise<void> {
  const response = await fetch('api/ai/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messages, llmConfig }),
    signal,
  });

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    try {
      const body = (await response.json()) as { Message?: string };
      if (body.Message) {
        message = body.Message;
      }
    } catch {
      // non-JSON error body; keep the default message
    }
    throw new Error(message);
  }

  if (!response.body) {
    throw new Error('No response body received from the AI service');
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });

    let separatorIndex = buffer.indexOf('\n\n');
    while (separatorIndex !== -1) {
      const block = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      separatorIndex = buffer.indexOf('\n\n');

      const sseEvent = parseSSEBlock(block);
      if (sseEvent) {
        dispatchSSEEvent(sseEvent, handlers);
      }
    }
  }
}
