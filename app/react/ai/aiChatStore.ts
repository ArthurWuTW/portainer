import create from 'zustand';
import { persist } from 'zustand/middleware';

import { keyBuilder } from '@/react/hooks/useLocalStorage';

import { streamChat } from './aiChatService';
import {
  type ChatMessage,
  type LLMConfig,
  type ToolEvent,
  DEFAULT_LLM_CONFIG,
} from './types';

const MIN_WIDTH = 320;
const MAX_WIDTH = 800;
const DEFAULT_WIDTH = 420;

interface AIChatState {
  isOpen: boolean;
  width: number;
  llmConfig: LLMConfig;

  messages: ChatMessage[];
  toolEvents: ToolEvent[];
  isStreaming: boolean;
  error: string | null;

  abortController: AbortController | null;

  toggle(): void;
  setOpen(open: boolean): void;
  setWidth(width: number): void;
  setLLMConfig(config: LLMConfig): void;
  clearConversation(): void;
  sendMessage(content: string): Promise<void>;
  stopStreaming(): void;
}

function clampWidth(width: number) {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, width));
}

export const useAIChatStore = create<AIChatState>()(
  persist(
    (set, get) => ({
      isOpen: false,
      width: DEFAULT_WIDTH,
      llmConfig: DEFAULT_LLM_CONFIG,

      messages: [],
      toolEvents: [],
      isStreaming: false,
      error: null,

      abortController: null,

      toggle: () => {
        set((state) => ({ isOpen: !state.isOpen }));
      },
      setOpen: (open) => {
        set({ isOpen: open });
      },
      setWidth: (width) => {
        set({ width: clampWidth(width) });
      },
      setLLMConfig: (config) => {
        set({ llmConfig: config });
      },
      clearConversation: () => {
        const { abortController } = get();
        abortController?.abort();
        set({
          messages: [],
          toolEvents: [],
          isStreaming: false,
          error: null,
          abortController: null,
        });
      },
      sendMessage: async (content) => {
        const { isStreaming, messages, llmConfig } = get();
        if (isStreaming || !content.trim()) {
          return;
        }

        const userMessage: ChatMessage = { role: 'user', content: content.trim() };
        const assistantMessage: ChatMessage = { role: 'assistant', content: '' };
        const history = [...messages, userMessage, assistantMessage];

        const abortController = new AbortController();
        set({
          messages: history,
          toolEvents: [],
          isStreaming: true,
          error: null,
          abortController,
        });

        function appendToLastAssistant(delta: string) {
          set((state) => {
            const msgs = [...state.messages];
            const last = msgs[msgs.length - 1];
            if (last && last.role === 'assistant') {
              msgs[msgs.length - 1] = { ...last, content: last.content + delta };
            }
            return { messages: msgs };
          });
        }

        try {
          await streamChat(
            history,
            llmConfig,
            {
              onToken: appendToLastAssistant,
              onToolStart: (name, arguments_) => {
                set((state) => ({
                  toolEvents: [
                    ...state.toolEvents,
                    { name, arguments: arguments_, status: 'running' },
                  ],
                }));
              },
              onToolEnd: (name, ok) => {
                set((state) => {
                  const events = [...state.toolEvents];
                  for (let i = events.length - 1; i >= 0; i--) {
                    if (events[i].name === name && events[i].status === 'running') {
                      events[i] = { ...events[i], status: ok ? 'done' : 'error' };
                      break;
                    }
                  }
                  return { toolEvents: events };
                });
              },
              onError: (message) => {
                set({ error: message, isStreaming: false, abortController: null });
              },
              onDone: () => {
                set({ isStreaming: false, abortController: null });
              },
            },
            abortController.signal
          );
        } catch (err) {
          const message =
            err instanceof Error ? err.message : 'Failed to reach the AI service';
          set({ error: message, isStreaming: false, abortController: null });
        }
      },
      stopStreaming: () => {
        const { abortController } = get();
        abortController?.abort();
        set({ isStreaming: false, abortController: null });
      },
    }),
    {
      name: keyBuilder('AI_CHAT'),
      partialize: (state) => ({
        isOpen: state.isOpen,
        width: state.width,
        llmConfig: state.llmConfig,
      }),
    }
  )
);
