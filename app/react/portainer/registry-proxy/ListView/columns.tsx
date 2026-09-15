import { createColumnHelper } from '@tanstack/react-table';

import { Badge } from '@@/Badge';
import { Link } from '@@/Link';

import { RegistryProxy } from '../types';

export const columnHelper = createColumnHelper<RegistryProxy>();

export function getColumns() {
  return [
    columnHelper.accessor('Name', {
      header: 'Name',
      cell: ({ getValue, row: { original: proxy } }) => (
        <Link
          to="portainer.registry-proxy.item"
          params={{ id: proxy.Id }}
          data-cy={`registry-proxy-link-${proxy.Id}`}
        >
          {getValue()}
        </Link>
      ),
    }),
    columnHelper.accessor('URL', {
      header: 'Local registry URL',
    }),
    columnHelper.accessor('TLS', {
      header: 'TLS',
      cell: ({ getValue }) => (
        <Badge type={getValue() ? 'success' : 'muted'}>
          {getValue() ? 'Enabled' : 'Disabled'}
        </Badge>
      ),
    }),
    columnHelper.accessor('Authentication', {
      header: 'Registry auth',
      cell: ({ getValue }) => (
        <Badge type={getValue() ? 'success' : 'muted'}>
          {getValue() ? 'Enabled' : 'Disabled'}
        </Badge>
      ),
    }),
  ];
}
