import { useState } from 'react';
import { createColumnHelper } from '@tanstack/react-table';

import { useIdParam } from '@/react/hooks/useIdParam';
import { notifySuccess } from '@/portainer/services/notifications';

import { PageHeader } from '@@/PageHeader';
import { DetailsTable } from '@@/DetailsTable';
import { Datatable } from '@@/datatables';
import { createPersistedStore } from '@@/datatables/types';
import { useTableState } from '@@/datatables/useTableState';
import { Alert } from '@@/Alert';
import { Button, LoadingButton } from '@@/buttons';
import { DeleteButton } from '@@/buttons/DeleteButton';
import { FormControl } from '@@/form-components/FormControl';
import { Input } from '@@/form-components/Input';
import { PortainerSelect } from '@@/form-components/PortainerSelect';

import { useRegistryProxy } from '../queries/useRegistryProxy';
import { useRegistryProxyCatalog } from '../queries/useRegistryProxyCatalog';
import { useRegistryProxyTags } from '../queries/useRegistryProxyTags';
import {
  useAddRegistryTag,
  useDeleteRegistryImage,
} from '../queries/useRegistryProxyMutations';

interface TagRow {
  Name: string;
}

const tagColumnHelper = createColumnHelper<TagRow>();

const tableKey = 'registry-proxy-tags';
const settingsStore = createPersistedStore(tableKey, 'Name');

export function ItemView() {
  const id = useIdParam('id');
  const proxyQuery = useRegistryProxy(id);
  const catalogQuery = useRegistryProxyCatalog(id);
  const [selectedRepository, setSelectedRepository] = useState<string | null>(
    null
  );
  const tagsQuery = useRegistryProxyTags(id, selectedRepository);
  const deleteImageMutation = useDeleteRegistryImage(id, selectedRepository);
  const addTagMutation = useAddRegistryTag(id);
  const tableState = useTableState(settingsStore, tableKey);

  const [retagSource, setRetagSource] = useState<string>('');
  const [retagTarget, setRetagTarget] = useState<string>('');

  const host = window.location.host;
  const proxyPath = `${host}/registry-proxy/${id}`;

  const proxy = proxyQuery.data;

  return (
    <>
      <PageHeader
        title={proxy?.Name ?? 'Registry proxy'}
        breadcrumbs={[
          {
            label: 'Registry Proxies',
            link: 'portainer.registry-proxy',
          },
          { label: proxy?.Name ?? 'Registry proxy' },
        ]}
        reload
      />

      {proxyQuery.error && (
        <div className="mx-4">
          <Alert color="error" title="Unable to load registry proxy">
            {(proxyQuery.error as Error).message}
          </Alert>
        </div>
      )}

      <div className="mx-4 mb-4 flex flex-col gap-4 lg:flex-row">
        <div className="w-full lg:w-1/3">
          <DetailsTable dataCy="registry-proxy-details">
            <DetailsTable.Row label="Name">
              {proxy?.Name ?? '-'}
            </DetailsTable.Row>
            <DetailsTable.Row label="Local registry URL">
              {proxy?.URL ?? '-'}
            </DetailsTable.Row>
            <DetailsTable.Row label="TLS">
              {proxy?.TLS ? 'Enabled' : 'Disabled'}
            </DetailsTable.Row>
            <DetailsTable.Row label="Registry authentication">
              {proxy?.Authentication
                ? `Enabled (${proxy.Username})`
                : 'Disabled'}
            </DetailsTable.Row>
            <DetailsTable.Row label="Proxy endpoint">
              <code>{`${window.location.protocol}//${proxyPath}/`}</code>
            </DetailsTable.Row>
          </DetailsTable>

          <div className="border-border mt-4 rounded border border-2 border-solid p-4 text-sm">
            <h5 className="mb-2 font-medium">How to use</h5>
            <p className="mb-1">
              Authenticate with your Portainer username and password:
            </p>
            <pre className="mb-2 whitespace-pre-wrap break-all">
              {`docker login ${host}`}
            </pre>
            <p className="mb-1">Pull an image through the proxy:</p>
            <pre className="mb-2 whitespace-pre-wrap break-all">
              {`docker pull ${proxyPath}/<image>:<tag>`}
            </pre>
            <p className="mb-1">Push an image (Portainer admins only):</p>
            <pre className="whitespace-pre-wrap break-all">
              {`docker push ${proxyPath}/<image>:<tag>`}
            </pre>
          </div>
        </div>

        <div className="flex w-full flex-col gap-4 lg:w-2/3">
          <div>
            <h4 className="mb-2">Image repositories</h4>
            {catalogQuery.isLoading && <span>Loading repositories...</span>}
            {catalogQuery.error && (
              <Alert color="error" title="Unable to load image list">
                {(catalogQuery.error as Error).message}
              </Alert>
            )}
            {catalogQuery.data &&
              catalogQuery.data.repositories.length === 0 && (
                <span>No images in this registry.</span>
              )}
            <div className="flex flex-wrap gap-2">
              {catalogQuery.data?.repositories.map((repository) => (
                <Button
                  key={repository}
                  size="small"
                  color="default"
                  className={
                    selectedRepository === repository
                      ? '!bg-[var(--primary-50)]'
                      : undefined
                  }
                  onClick={() => setSelectedRepository(repository)}
                  data-cy={`registry-proxy-repository-${repository}`}
                >
                  {repository}
                </Button>
              ))}
            </div>
          </div>

          {selectedRepository && (
            <>
              <div className="flex flex-wrap items-end gap-4">
                <FormControl
                  label="Tag existing image with a new version"
                  inputId="retag-source"
                  className="min-w-[200px]"
                >
                  <PortainerSelect
                    inputId="retag-source"
                    name="retag-source"
                    value={retagSource || undefined}
                    onChange={(value) =>
                      setRetagSource((value as string) ?? '')
                    }
                    options={(tagsQuery.data?.tags ?? []).map((tag) => ({
                      value: tag,
                      label: tag,
                    }))}
                    placeholder="Source tag"
                    isLoading={tagsQuery.isLoading}
                    data-cy="registry-proxy-retag-source-select"
                  />
                </FormControl>
                <FormControl label="New tag" inputId="retag-target">
                  <Input
                    id="retag-target"
                    name="retag-target"
                    value={retagTarget}
                    onChange={(e) => setRetagTarget(e.target.value)}
                    placeholder="e.g. latest"
                    data-cy="registry-proxy-retag-target-input"
                  />
                </FormControl>
                <LoadingButton
                  size="medium"
                  color="primary"
                  loadingText="Tagging..."
                  isLoading={addTagMutation.isLoading}
                  disabled={!retagSource || !retagTarget}
                  onClick={handleAddTag}
                  data-cy="registry-proxy-retag-button"
                >
                  Add tag
                </LoadingButton>
              </div>

              <Datatable
                title={`Versions of ${selectedRepository}`}
                dataset={(tagsQuery.data?.tags ?? []).map((tag) => ({
                  Name: tag,
                }))}
                columns={[
                  tagColumnHelper.accessor('Name', { header: 'Version (tag)' }),
                  tagColumnHelper.display({
                    id: 'actions',
                    header: 'Actions',
                    cell: ({ row: { original: tag } }) => (
                      <DeleteButton
                        size="small"
                        text="Delete"
                        loadingText="Deleting..."
                        confirmMessage={`This will remove '${selectedRepository}:${tag.Name}' from the registry. Continue?`}
                        onConfirmed={() =>
                          deleteImageMutation.mutateAsync(tag.Name)
                        }
                        data-cy={`registry-proxy-delete-${tag.Name}`}
                      />
                    ),
                  }),
                ]}
                settingsManager={tableState}
                getRowId={(row) => row.Name}
                isLoading={tagsQuery.isLoading}
                emptyContentLabel="No image versions"
                data-cy="registry-proxy-tags-datatable"
              />
            </>
          )}
        </div>
      </div>
    </>
  );

  async function handleAddTag() {
    if (!selectedRepository) {
      return;
    }

    await addTagMutation.mutateAsync(
      {
        repository: selectedRepository,
        source: retagSource,
        target: retagTarget,
      },
      {
        onSuccess: () => {
          notifySuccess('Success', 'Image successfully tagged');
          setRetagTarget('');
        },
      }
    );
  }
}
