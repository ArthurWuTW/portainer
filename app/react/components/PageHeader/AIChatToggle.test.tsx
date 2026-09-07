import { fireEvent, render, screen } from '@testing-library/react';

import { useAIChatStore } from '@/react/ai/aiChatStore';

import { AIChatToggle } from './AIChatToggle';

beforeEach(() => {
  useAIChatStore.setState({ isOpen: false });
});

describe('AIChatToggle', () => {
  test('opens the chat panel when clicked', () => {
    render(<AIChatToggle />);

    fireEvent.click(screen.getByTestId('ai-chat-toggle'));

    expect(useAIChatStore.getState().isOpen).toBe(true);
  });

  test('closes the chat panel when clicked while open', () => {
    useAIChatStore.setState({ isOpen: true });
    render(<AIChatToggle />);

    fireEvent.click(screen.getByTestId('ai-chat-toggle'));

    expect(useAIChatStore.getState().isOpen).toBe(false);
  });
});
