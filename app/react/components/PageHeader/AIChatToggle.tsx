import { BotMessageSquare } from 'lucide-react';
import clsx from 'clsx';

import { useAIChatStore } from '@/react/ai/aiChatStore';

import headerStyles from './HeaderTitle.module.css';

export function AIChatToggle() {
  const { isOpen, toggle } = useAIChatStore();

  return (
    <div className={headerStyles.menuButton}>
      <button
        type="button"
        className={clsx(
          headerStyles.menuIcon,
          'icon-badge mr-1 cursor-pointer !p-2 text-lg',
          'text-gray-8',
          'th-dark:text-gray-warm-7',
          { 'bg-blue-1/10': isOpen }
        )}
        title="AI Assistant"
        data-cy="ai-chat-toggle"
        onClick={toggle}
      >
        <BotMessageSquare className="lucide" />
      </button>
    </div>
  );
}
