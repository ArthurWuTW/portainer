import { createColumnHelper } from '@tanstack/react-table';
import { ArrowLeft } from 'lucide-react';

import { localizeDate } from '@/react/common/date-utils';
import { humanize } from '@/portainer/filters/filters';
import { trimSHA } from '@/docker/filters/utils';

import { Datatable } from '@@/datatables';
import { createPersistedStore } from '@@/datatables/types';
import { useTableState } from '@@/datatables/useTableState';
import { Button } from '@@/buttons';
import { DeleteButton } from '@@/buttons/DeleteButton';
import { Link } from '@@/Link';

import { RegistryProxyId } from '../types';
import { useRegistryProxyTags } from '../queries/useRegistryProxyTags';
import { useDeleteRegistryImage } from '../queries/useRegistryProxyMutations';

interface TagRow {
  Name: string;
  Digest: string;
  Size: number;
  Created?: string;
}

const columnHelper = createColumnHelper<TagRow>();

const tableKey = 'registry-proxy-tags';
const settingsStore = createPersistedStore(tableKey, 'Name');

interface Props {
  id: RegistryProxyId;
  repository: string;
  onBack: () => void;
}

export function RepositoryView({ id, repository, onBack }: Props) {
  const tagsQuery = useRegistryProxyTags(id, repository);
  const deleteImageMutation = useDeleteRegistryImage(id, repository);
  const tableState = useTableState(settingsStore, tableKey);

  return (
    <Datatable
      title={`Versions of ${repository}`}
      dataset={(tagsQuery.data?.tags ?? []).map((tag) => ({
        Name: tag.name,
        Digest: tag.digest,
        Size: tag.size,
        Created: tag.created,
      }))}
      columns={[
        columnHelper.accessor('Name', {
          header: 'Version (tag)',
          cell: ({ row: { original: tag } }) => (
            <Link
              to="."
              params={{ repository, tag: tag.Name }}
              data-cy={`registry-proxy-tag-link-${tag.Name}`}
            >
              {tag.Name}
            </Link>
          ),
        }),
        columnHelper.accessor('Digest', {
          header: 'Digest',
          cell: ({ getValue }) => (
            <span title={getValue()}>{trimSHA(getValue())}</span>
          ),
        }),
        columnHelper.accessor('Size', {
          header: 'Manifest size',
          cell: ({ getValue }) => humanize(getValue()),
        }),
        columnHelper.accessor('Created', {
          header: 'Created',
          sortingFn: 'datetime',
          cell: ({ getValue }) => {
            const created = getValue();

            return created ? localizeDate(new Date(created)) : '-';
          },
        }),
        columnHelper.display({
          id: 'actions',
          header: 'Actions',
          cell: ({ row: { original: tag } }) => (
            <DeleteButton
              size="small"
              text="Delete"
              loadingText="Deleting..."
              confirmMessage={`This will remove '${repository}:${tag.Name}' from the registry. Continue?`}
              onConfirmed={() => deleteImageMutation.mutate(tag.Name)}
              data-cy={`registry-proxy-delete-${tag.Name}`}
            />
          ),
        }),
      ]}
      settingsManager={tableState}
      getRowId={(row) => row.Name}
      isLoading={tagsQuery.isLoading}
      emptyContentLabel="No image versions"
      renderTableActions={() => (
        <Button
          color="link"
          icon={ArrowLeft}
          className="!m-0"
          data-cy="registry-proxy-back-button"
          onClick={onBack}
        >
          Back to registry
        </Button>
      )}
      data-cy="registry-proxy-tags-datatable"
    />
  );
}
