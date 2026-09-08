import {
  BotMessageSquare,
  Loader2,
  Send,
  Settings2,
  Square,
  Trash2,
  X,
} from 'lucide-react';
import Markdown from 'markdown-to-jsx';
import { useCallback, useEffect, useRef, useState } from 'react';

import { Button } from '@@/buttons';

import { useAIChatStore } from './aiChatStore';
import { LLMSettingsForm } from './LLMSettingsForm';
import type { ChatMessage, ToolEvent } from './types';
import './AIChatPanel.css';

export function AIChatPanel() {
  const { isOpen, width } = useAIChatStore();

  if (!isOpen) {
    return null;
  }

  return <InnerPanel width={width} />;
}

function InnerPanel({ width }: { width: number }) {
  const {
    messages,
    toolEvents,
    isStreaming,
    error,
    setWidth,
    setOpen,
    clearConversation,
    sendMessage,
    stopStreaming,
  } = useAIChatStore();

  const [input, setInput] = useState('');
  const [showSettings, setShowSettings] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to the bottom as new content arrives.
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages, toolEvents, isStreaming]);

  const handleResizeStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startWidth = useAIChatStore.getState().width;

      function onMove(ev: MouseEvent) {
        setWidth(startWidth + (startX - ev.clientX));
      }
      function onUp() {
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }

      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [setWidth]
  );

  const handleSend = useCallback(() => {
    if (!input.trim() || isStreaming) {
      return;
    }
    const content = input;
    setInput('');
    void sendMessage(content);
  }, [input, isStreaming, sendMessage]);

  return (
    <div
      data-cy="ai-chat-panel"
      className="bg-widget-color th-dark:bg-gray-12 fixed inset-y-0 right-0 z-[60] flex flex-col border-l border-gray-3 shadow-lg th-dark:border-gray-11"
      style={{ width }}
    >
      {/* Resize handle */}
      <button
        type="button"
        data-cy="ai-chat-resize-handle"
        aria-label="Resize AI chat panel"
        onMouseDown={handleResizeStart}
        className="absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize border-0 bg-transparent p-0"
      />

      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-3 px-4 py-3 th-dark:border-gray-11">
        <div className="flex items-center gap-2">
          <BotMessageSquare className="h-5 w-5 text-gray-8 th-dark:text-gray-warm-7" />
          <span className="text-sm font-semibold text-gray-10 th-dark:text-gray-1">
            AI Assistant
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            color="none"
            size="small"
            icon={Settings2}
            title="AI settings"
            data-cy="ai-chat-settings-button"
            onClick={() => setShowSettings((v) => !v)}
          />
          <Button
            color="none"
            size="small"
            icon={Trash2}
            title="Clear conversation"
            data-cy="ai-chat-clear-button"
            onClick={clearConversation}
          />
          <Button
            color="none"
            size="xsmall"
            icon={X}
            title="Close"
            data-cy="ai-chat-close-button"
            onClick={() => setOpen(false)}
          />
        </div>
      </div>

      {/* Settings */}
      {showSettings && (
        <div className="border-b border-gray-3 px-4 py-3 th-dark:border-gray-11">
          <LLMSettingsForm />
        </div>
      )}

      {/* Messages */}
      <div
        ref={scrollRef}
        data-cy="ai-chat-messages"
        className="flex-1 overflow-y-auto px-4 py-3"
      >
        {messages.length === 0 && !isStreaming ? (
          <div className="mt-8 text-center text-sm text-gray-8 th-dark:text-gray-warm-7">
            Ask about your environments, stacks, and containers.
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map((msg, i) => (
              <MessageBubble
                key={i}
                message={msg}
                animate={i === messages.length - 1 && msg.role === 'assistant'}
              />
            ))}
            {toolEvents.length > 0 && (
              <div className="space-y-1">
                {toolEvents.map((tool, i) => (
                  <ToolChip key={i} tool={tool} />
                ))}
              </div>
            )}
            {isStreaming && (
              <div className="flex items-center gap-2 text-xs text-gray-8 th-dark:text-gray-warm-7">
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                <span>Thinking…</span>
              </div>
            )}
          </div>
        )}
        {error && (
          <div
            data-cy="ai-chat-error"
            className="border-danger-1 bg-danger-1/10 text-danger-1 th-dark:border-danger-1 th-dark:bg-danger-1/10 th-dark:text-danger-1 mt-3 rounded border px-3 py-2 text-xs"
          >
            {error}
          </div>
        )}
      </div>

      {/* Input */}
      <div className="border-t border-gray-3 px-4 py-3 th-dark:border-gray-11">
        <div className="flex items-end gap-2">
          <textarea
            data-cy="ai-chat-input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
            }}
            rows={2}
            placeholder="Ask about your Portainer instance…"
            className="bg-widget-color th-dark:bg-gray-12 flex-1 resize-none rounded border border-gray-3 px-3 py-2 text-sm text-gray-10 outline-none focus:border-blue-1 th-dark:border-gray-11 th-dark:text-gray-1 th-dark:focus:border-blue-1"
          />
          {isStreaming ? (
            <Button
              color="secondary"
              size="medium"
              icon={Square}
              title="Stop"
              data-cy="ai-chat-stop-button"
              onClick={stopStreaming}
            />
          ) : (
            <Button
              color="primary"
              size="medium"
              icon={Send}
              title="Send"
              data-cy="ai-chat-send-button"
              disabled={!input.trim()}
              onClick={handleSend}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function useTypewriter(text: string, enabled: boolean) {
  const [displayed, setDisplayed] = useState('');

  useEffect(() => {
    if (!enabled || displayed.length >= text.length) {
      return;
    }

    const id = setTimeout(() => {
      if (!text.startsWith(displayed)) {
        setDisplayed('');
        return;
      }
      const remaining = text.slice(displayed.length);
      const nextChunk =
        remaining.match(/^\s*\S+\s*/)?.[0] ?? remaining.slice(0, 1);
      setDisplayed(displayed + nextChunk);
    }, 30);

    return () => clearTimeout(id);
  }, [displayed, text, enabled]);

  if (!enabled) {
    return text;
  }

  return displayed;
}

function MessageBubble({
  message,
  animate,
}: {
  message: ChatMessage;
  animate: boolean;
}) {
  const isUser = message.role === 'user';
  const content = useTypewriter(message.content, animate && !isUser);

  return (
    <div className={isUser ? 'flex justify-end' : 'flex justify-start'}>
      <div
        data-cy={isUser ? 'ai-chat-user-message' : 'ai-chat-assistant-message'}
        className={
          isUser
            ? 'max-w-[85%] rounded-lg bg-blue-1 px-3 py-2 text-sm text-black'
            : 'max-w-[85%] rounded-lg bg-gray-2 px-3 py-2 text-sm text-gray-10 th-dark:bg-gray-11 th-dark:text-gray-1'
        }
      >
        {content ? (
          <Markdown className="ai-chat-markdown">{content}</Markdown>
        ) : isUser ? null : (
          '…'
        )}
      </div>
    </div>
  );
}

function ToolChip({ tool }: { tool: ToolEvent }) {
  const statusColor =
    tool.status === 'running'
      ? 'text-blue-1 th-dark:text-blue-1'
      : tool.status === 'done'
        ? 'text-success-7 th-dark:text-success-3'
        : 'text-danger-1 th-dark:text-danger-1';

  return (
    <div
      data-cy="ai-chat-tool-event"
      className="flex items-center gap-2 rounded bg-gray-2 px-2 py-1 text-xs text-gray-8 th-dark:bg-gray-11 th-dark:text-gray-warm-7"
    >
      {tool.status === 'running' ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <span className={statusColor}>
          {tool.status === 'done' ? '✓' : '✗'}
        </span>
      )}
      <span className="font-mono">{tool.name}</span>
    </div>
  );
}
