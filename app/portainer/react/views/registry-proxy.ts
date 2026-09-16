import angular from 'angular';

import { ListView } from '@/react/portainer/registry-proxy/ListView/ListView';
import { ItemView } from '@/react/portainer/registry-proxy/ItemView/ItemView';
import { CreateView } from '@/react/portainer/registry-proxy/CreateView/CreateView';
import { EditView } from '@/react/portainer/registry-proxy/EditView/EditView';
import { r2a } from '@/react-tools/react2angular';
import { withCurrentUser } from '@/react-tools/withCurrentUser';
import { withReactQuery } from '@/react-tools/withReactQuery';
import { withUIRouter } from '@/react-tools/withUIRouter';

export const registryProxyModule = angular
  .module('portainer.app.react.views.registry-proxy', [])
  .component(
    'registryProxyListView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(ListView))), [])
  )
  .component(
    'registryProxyItemView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(ItemView))), [])
  )
  .component(
    'registryProxyCreateView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(CreateView))), [])
  )
  .component(
    'registryProxyEditView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(EditView))), [])
  ).name;
