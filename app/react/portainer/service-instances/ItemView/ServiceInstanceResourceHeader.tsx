import { Boxes } from 'lucide-react';

import { Badge } from '@@/Badge';
import { Icon } from '@@/Icon';
import { ResourceDetailHeader } from '@@/ResourceDetailHeader/ResourceDetailHeader';

import { ServiceInstance, ServiceInstanceStatuses } from '../types';

const statusBadgeType: Record<
  number,
  'success' | 'danger' | 'warn' | 'info' | 'muted'
> = {
  [ServiceInstanceStatuses.RUNNING]: 'success',
  [ServiceInstanceStatuses.STOPPED]: 'muted',
  [ServiceInstanceStatuses.DEPLOYING]: 'info',
  [ServiceInstanceStatuses.PARTIAL]: 'warn',
  [ServiceInstanceStatuses.FAILED]: 'danger',
  [ServiceInstanceStatuses.UNKNOWN]: 'muted',
};

const statusLabel: Record<number, string> = {
  [ServiceInstanceStatuses.RUNNING]: 'Running',
  [ServiceInstanceStatuses.STOPPED]: 'Stopped',
  [ServiceInstanceStatuses.DEPLOYING]: 'Deploying',
  [ServiceInstanceStatuses.PARTIAL]: 'Partial',
  [ServiceInstanceStatuses.FAILED]: 'Failed',
  [ServiceInstanceStatuses.UNKNOWN]: 'Unknown',
};

interface Props {
  instance: ServiceInstance;
}

export function ServiceInstanceResourceHeader({ instance }: Props) {
  return (
    <ResourceDetailHeader
      icon={<Icon icon={Boxes} size="xl" />}
      title={instance.Name}
      badge={
        <Badge type={statusBadgeType[instance.Status] ?? 'muted'}>
          {statusLabel[instance.Status] ?? 'Unknown'}
        </Badge>
      }
      description={instance.Description || undefined}
    />
  );
}
