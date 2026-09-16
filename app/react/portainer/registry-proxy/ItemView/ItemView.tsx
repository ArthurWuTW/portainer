import { Pencil } from 'lucide-react';

import { useIdParam } from '@/react/hooks/useIdParam';

import { PageHeader } from '@@/PageHeader';
import { Alert } from '@@/Alert';
import { Button } from '@@/buttons';
import { Link } from '@@/Link';

import { useRegistryProxy } from '../queries/useRegistryProxy';
import { useRegistryProxyCatalog } from '../queries/useRegistryProxyCatalog';
import { useRegistryProxyTags } from '../queries/useRegistryProxyTags';
import { useImageNavigation } from '../useImageNavigation';
import { RepositoryTree } from '../RepositoryTree/RepositoryTree';

import { ProxyDetailsPanel } from './ProxyDetailsPanel';
import { RepositoryView } from './RepositoryView';
import { TagView } from './TagView';

export function ItemView() {
  const id = useIdParam('id');
  const proxyQuery = useRegistryProxy(id);
  const catalogQuery = useRegistryProxyCatalog(id);
  const [{ repository, tag }, navigation] = useImageNavigation();
  const tagsQuery = useRegistryProxyTags(id, repository);

  const proxy = proxyQuery.data;
  const repositories = catalogQuery.data?.repositories ?? [];
  const proxyPath = `${window.location.host}/registry-proxy/${id}`;

  return (
    <>
      <PageHeader
        title={getImageTitle(proxy?.Name, repository, tag)}
        breadcrumbs={[
          {
            label: 'Registry Proxies',
            link: 'portainer.registry-proxy',
          },
          {
            label: proxy?.Name ?? 'Registry proxy',
            link: repository ? 'portainer.registry-proxy.item' : undefined,
            linkParams: { id },
          },
          ...(repository
            ? [
                {
                  label: repository,
                  link: tag ? 'portainer.registry-proxy.item' : undefined,
                  linkParams: { id, repository },
                },
              ]
            : []),
          ...(tag ? [{ label: tag }] : []),
        ]}
        reload
      >
        <Button
          color="primary"
          size="large"
          icon={Pencil}
          className="!m-0"
          as={Link}
          props={{
            to: 'portainer.registry-proxy.item.edit',
            params: { id },
          }}
          data-cy="registry-proxy-edit-button"
        >
          Edit
        </Button>
      </PageHeader>

      {proxyQuery.error && (
        <div className="mx-4">
          <Alert color="error" title="Unable to load registry proxy">
            {(proxyQuery.error as Error).message}
          </Alert>
        </div>
      )}

      <div className="mx-4 mb-4 flex flex-col gap-4 lg:flex-row">
        <div className="w-full lg:w-1/4">
          <h4 className="mb-2">Image repositories</h4>
          {catalogQuery.isLoading && <span>Loading repositories...</span>}
          {catalogQuery.error && (
            <Alert color="error" title="Unable to load image list">
              {(catalogQuery.error as Error).message}
            </Alert>
          )}
          {catalogQuery.data && repositories.length === 0 && (
            <span>No images in this registry.</span>
          )}
          {repositories.length > 0 && (
            <RepositoryTree
              repositories={repositories}
              selection={{ repository, tag }}
              tags={tagsQuery.data?.tags ?? []}
              isLoadingTags={tagsQuery.isLoading}
              onOpenRepository={navigation.openRepository}
              onOpenTag={navigation.openTag}
            />
          )}
        </div>

        <div className="w-full lg:w-3/4">
          {repository && tag ? (
            <TagView
              id={id}
              repository={repository}
              tag={tag}
              proxyPath={proxyPath}
              onBack={() => navigation.openRepository(repository)}
              onDeleted={() => {
                // Once the repository's last tag is gone, the repository
                // disappears from the registry, so drop back to the proxy.
                if ((tagsQuery.data?.tags.length ?? 0) <= 1) {
                  navigation.openProxy();
                } else {
                  navigation.openRepository(repository);
                }
              }}
            />
          ) : repository ? (
            <RepositoryView
              id={id}
              repository={repository}
              onBack={navigation.openProxy}
            />
          ) : (
            <ProxyDetailsPanel proxy={proxy} proxyPath={proxyPath} />
          )}
        </div>
      </div>
    </>
  );
}

function getImageTitle(
  proxyName: string | undefined,
  repository: string | null,
  tag: string | null
) {
  if (repository && tag) {
    return `${repository}:${tag}`;
  }

  return repository ?? proxyName ?? 'Registry proxy';
}
