import { Network, Trash2 } from 'lucide-react';

import { Datatable } from '@@/datatables';
import { createPersistedStore } from '@@/datatables/types';
import { AddButton, Button } from '@@/buttons';
import { useTableState } from '@@/datatables/useTableState';
import { confirmDelete } from '@@/modals/confirm';
import { PageHeader } from '@@/PageHeader';

import { useRegistryProxies } from '../queries/useRegistryProxies';
import { useDeleteRegistryProxy } from '../queries/useDeleteRegistryProxy';
import { RegistryProxy } from '../types';

import { getColumns } from './columns';

const tableKey = 'registry-proxies';
const settingsStore = createPersistedStore(tableKey, 'Name');

export function ListView() {
  const tableState = useTableState(settingsStore, tableKey);
  const { data: proxies, isLoading } = useRegistryProxies();
  const deleteMutation = useDeleteRegistryProxy();

  return (
    <>
      <PageHeader
        title="Registry Proxies"
        breadcrumbs="Registry Proxies"
        reload
      />
      <div className="mx-4 mb-4">
        <Datatable
          title="Registry Proxies"
          titleIcon={Network}
          dataset={proxies ?? []}
          columns={getColumns()}
          settingsManager={tableState}
          isLoading={isLoading}
          emptyContentLabel="No registry proxies"
          renderTableActions={(selectedRows) => (
            <div className="flex items-center gap-2">
              <Button
                color="dangerlight"
                disabled={selectedRows.length === 0}
                icon={Trash2}
                className="!m-0"
                data-cy="remove-registry-proxies-button"
                onClick={() => handleRemove(selectedRows)}
              >
                Remove
              </Button>
              <AddButton
                to="portainer.registry-proxy.new"
                data-cy="registry-proxies-add-button"
              >
                Register local registry
              </AddButton>
            </div>
          )}
          data-cy="registry-proxies-datatable"
        />
      </div>
    </>
  );

  async function handleRemove(rows: RegistryProxy[]) {
    const confirmed = await confirmDelete(
      'This action will remove the selected registry proxy/proxies. The proxied registries themselves are not modified. Continue?'
    );
    if (!confirmed) {
      return;
    }

    for (const row of rows) {
      await deleteMutation.mutateAsync(row.Id);
    }
  }
}
