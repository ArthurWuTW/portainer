import angular from 'angular';

import { AIChatPanel } from '@/react/ai/AIChatPanel';
import { r2a } from '@/react-tools/react2angular';
import { withCurrentUser } from '@/react-tools/withCurrentUser';
import { withReactQuery } from '@/react-tools/withReactQuery';
import { withUIRouter } from '@/react-tools/withUIRouter';

export const aiChatModule = angular
  .module('portainer.app.ai-chat', [])
  .component('aiChat', r2a(withUIRouter(withReactQuery(withCurrentUser(AIChatPanel))), []))
  .name;
