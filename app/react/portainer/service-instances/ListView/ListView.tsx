import { Boxes, Trash2 } from 'lucide-react';

import { Datatable } from '@@/datatables';
import { createPersistedStore } from '@@/datatables/types';
import { AddButton, Button } from '@@/buttons';
import { useTableState } from '@@/datatables/useTableState';
import { confirmDelete } from '@@/modals/confirm';
import { PageHeader } from '@@/PageHeader';

import { useServiceInstances } from '../queries/useServiceInstances';
import { useDeleteServiceInstance } from '../queries/useDeleteServiceInstance';
import { ServiceInstance } from '../types';

import { columns } from './columns';

const tableKey = 'service-instances';
const settingsStore = createPersistedStore(tableKey, 'Name');

export function ListView() {
  const tableState = useTableState(settingsStore, tableKey);
  const { data: instances, isLoading } = useServiceInstances();
  const deleteMutation = useDeleteServiceInstance();

  return (
    <>
      <PageHeader
        title="Service Instances"
        breadcrumbs="Service Instances"
        reload
      />
      <div className="mx-4 mb-4">
        <Datatable
          title="Service Instances"
          titleIcon={Boxes}
          dataset={instances ?? []}
          columns={columns}
          settingsManager={tableState}
          isLoading={isLoading}
          emptyContentLabel="No service instances"
          renderTableActions={(selectedRows) => (
            <div className="flex items-center gap-2">
              <Button
                color="dangerlight"
                disabled={selectedRows.length === 0}
                icon={Trash2}
                className="!m-0"
                data-cy="remove-service-instances-button"
                onClick={() => handleRemove(selectedRows)}
              >
                Remove
              </Button>
              <AddButton
                to="portainer.service-instances.new"
                data-cy="service-instances-add-button"
              >
                Add service instance
              </AddButton>
            </div>
          )}
          data-cy="service-instances-datatable"
        />
      </div>
    </>
  );

  async function handleRemove(rows: ServiceInstance[]) {
    const confirmed = await confirmDelete(
      'This action will remove the selected service instance(s). Stacks deployed on target environments are kept. Continue?'
    );
    if (!confirmed) {
      return;
    }

    for (const row of rows) {
      await deleteMutation.mutateAsync(row.Id);
    }
  }
}
