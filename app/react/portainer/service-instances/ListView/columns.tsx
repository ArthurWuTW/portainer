import { createColumnHelper } from '@tanstack/react-table';

import { Badge } from '@@/Badge';
import { Link } from '@@/Link';

import { ServiceInstance, ServiceInstanceStatuses } from '../types';

export const columnHelper = createColumnHelper<ServiceInstance>();

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

interface GetColumnsOptions {
  environmentNames: Map<number, string>;
  groupNames: Map<number, string>;
}

export function getColumns({
  environmentNames,
  groupNames,
}: GetColumnsOptions) {
  return [
    columnHelper.accessor('Name', {
      header: 'Name',
      cell: ({ getValue, row: { original: instance } }) => (
        <Link
          to="portainer.service-instances.item"
          params={{ id: instance.Id }}
          data-cy={`service-instance-link-${instance.Id}`}
        >
          {getValue()}
        </Link>
      ),
    }),
    columnHelper.accessor('TargetType', {
      header: 'Target',
      cell: ({ getValue, row: { original: instance } }) => {
        if (getValue() === 1) {
          return (
            groupNames.get(instance.GroupId ?? 0) ??
            `Group #${instance.GroupId}`
          );
        }
        const names = (instance.EnvironmentIds ?? []).map(
          (id) => environmentNames.get(id) ?? `#${id}`
        );
        return names.length > 0 ? names.join(', ') : '-';
      },
    }),
    columnHelper.accessor('Status', {
      header: 'Status',
      cell: ({ getValue }) => (
        <Badge type={statusBadgeType[getValue()] ?? 'muted'}>
          {statusLabel[getValue()] ?? 'Unknown'}
        </Badge>
      ),
    }),
    columnHelper.accessor('StackName', {
      header: 'Stack',
    }),
  ];
}
