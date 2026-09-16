import { createColumnHelper } from '@tanstack/react-table';
import { Pencil } from 'lucide-react';

import { Badge } from '@@/Badge';
import { Link } from '@@/Link';
import { Button } from '@@/buttons';

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
    columnHelper.display({
      id: 'actions',
      header: 'Actions',
      cell: ({ row: { original: proxy } }) => (
        <Button
          color="link"
          icon={Pencil}
          as={Link}
          props={{
            to: 'portainer.registry-proxy.item.edit',
            params: { id: proxy.Id },
          }}
          data-cy={`registry-proxy-edit-${proxy.Id}`}
        >
          Edit
        </Button>
      ),
    }),
  ];
}
