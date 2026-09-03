import angular from 'angular';

import { ListView } from '@/react/portainer/service-instances/ListView/ListView';
import { ItemView } from '@/react/portainer/service-instances/ItemView/ItemView';
import { CreateView } from '@/react/portainer/service-instances/CreateView/CreateView';
import { EditView } from '@/react/portainer/service-instances/EditView/EditView';
import { r2a } from '@/react-tools/react2angular';
import { withCurrentUser } from '@/react-tools/withCurrentUser';
import { withReactQuery } from '@/react-tools/withReactQuery';
import { withUIRouter } from '@/react-tools/withUIRouter';

export const serviceInstancesModule = angular
  .module('portainer.app.react.views.service-instances', [])
  .component(
    'serviceInstancesListView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(ListView))), [])
  )
  .component(
    'serviceInstanceItemView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(ItemView))), [])
  )
  .component(
    'serviceInstanceCreateView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(CreateView))), [])
  )
  .component(
    'serviceInstanceEditView',
    r2a(withUIRouter(withReactQuery(withCurrentUser(EditView))), [])
  ).name;
