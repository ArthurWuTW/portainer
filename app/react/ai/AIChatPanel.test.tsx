import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import { AIChatPanel } from './AIChatPanel';
import { useAIChatStore } from './aiChatStore';
import { streamChat } from './aiChatService';

vi.mock('./aiChatService', () => ({
  streamChat: vi.fn(),
}));

const mockedStreamChat = vi.mocked(streamChat);

function resetStore() {
  useAIChatStore.setState({
    isOpen: false,
    width: 420,
    messages: [],
    toolEvents: [],
    isStreaming: false,
    error: null,
    abortController: null,
  });
}

beforeEach(() => {
  resetStore();
  vi.clearAllMocks();
});

describe('AIChatPanel', () => {
  test('renders nothing when closed', () => {
    useAIChatStore.setState({ isOpen: false });
    render(<AIChatPanel />);
    expect(screen.queryByTestId('ai-chat-panel')).toBeNull();
  });

  test('renders the panel when open', () => {
    useAIChatStore.setState({ isOpen: true });
    render(<AIChatPanel />);
    expect(screen.getByTestId('ai-chat-panel')).toBeTruthy();
    expect(screen.getByTestId('ai-chat-input')).toBeTruthy();
  });

  test('close button closes the panel', () => {
    useAIChatStore.setState({ isOpen: true });
    render(<AIChatPanel />);

    fireEvent.click(screen.getByTestId('ai-chat-close-button'));

    expect(useAIChatStore.getState().isOpen).toBe(false);
  });

  test('sends a message and streams the assistant response', async () => {
    useAIChatStore.setState({ isOpen: true });

    mockedStreamChat.mockImplementation(async (_messages, _config, handlers) => {
      handlers.onToken('Hello ');
      handlers.onToken('world');
      handlers.onDone();
    });

    render(<AIChatPanel />);

    fireEvent.change(screen.getByTestId('ai-chat-input'), {
      target: { value: 'hi there' },
    });
    fireEvent.click(screen.getByTestId('ai-chat-send-button'));

    await waitFor(() => {
      expect(mockedStreamChat).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(screen.getByTestId('ai-chat-user-message')).toHaveTextContent(
        'hi there'
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('ai-chat-assistant-message')).toHaveTextContent(
        'Hello world'
      );
    });

    await waitFor(() => {
      expect(useAIChatStore.getState().isStreaming).toBe(false);
    });
  });

  test('displays tool execution status', async () => {
    useAIChatStore.setState({ isOpen: true });

    mockedStreamChat.mockImplementation(async (_messages, _config, handlers) => {
      handlers.onToolStart('get_endpoints', '{}');
      handlers.onToolEnd('get_endpoints', true);
      handlers.onToken('done');
      handlers.onDone();
    });

    render(<AIChatPanel />);

    fireEvent.change(screen.getByTestId('ai-chat-input'), {
      target: { value: 'list endpoints' },
    });
    fireEvent.click(screen.getByTestId('ai-chat-send-button'));

    await waitFor(() => {
      const tool = screen.getByTestId('ai-chat-tool-event');
      expect(tool).toHaveTextContent('get_endpoints');
    });

    await waitFor(() => {
      expect(useAIChatStore.getState().toolEvents[0].status).toBe('done');
    });
  });

  test('displays an error when the stream fails', async () => {
    useAIChatStore.setState({ isOpen: true });

    mockedStreamChat.mockImplementation(async (_messages, _config, handlers) => {
      handlers.onError('LLM request failed with status 500');
    });

    render(<AIChatPanel />);

    fireEvent.change(screen.getByTestId('ai-chat-input'), {
      target: { value: 'hi' },
    });
    fireEvent.click(screen.getByTestId('ai-chat-send-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ai-chat-error')).toHaveTextContent(
        'LLM request failed with status 500'
      );
    });

    await waitFor(() => {
      expect(useAIChatStore.getState().isStreaming).toBe(false);
    });
  });

  test('resizes the panel by dragging the handle', () => {
    useAIChatStore.setState({ isOpen: true, width: 420 });
    render(<AIChatPanel />);

    const handle = screen.getByTestId('ai-chat-resize-handle');
    const panel = screen.getByTestId('ai-chat-panel');

    fireEvent.mouseDown(handle, { clientX: 500 });
    fireEvent.mouseMove(document, { clientX: 400 });
    fireEvent.mouseUp(document);

    expect(useAIChatStore.getState().width).toBe(520);
    expect(panel).toHaveStyle({ width: '520px' });
  });

  test('clears the conversation', () => {
    useAIChatStore.setState({
      isOpen: true,
      messages: [
        { role: 'user', content: 'hello' },
        { role: 'assistant', content: 'hi' },
      ],
    });
    render(<AIChatPanel />);

    expect(screen.getByTestId('ai-chat-user-message')).toBeTruthy();

    fireEvent.click(screen.getByTestId('ai-chat-clear-button'));

    expect(useAIChatStore.getState().messages).toEqual([]);
  });
});
