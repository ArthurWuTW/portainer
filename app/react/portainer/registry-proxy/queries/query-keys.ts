import { RegistryProxyId } from '../types';

export const registryProxyQueryKeys = {
  base: () => ['registry-proxies'] as const,
  list: () => [...registryProxyQueryKeys.base()] as const,
  item: (id: RegistryProxyId) =>
    [...registryProxyQueryKeys.base(), id] as const,
  catalog: (id: RegistryProxyId) =>
    [...registryProxyQueryKeys.base(), id, 'catalog'] as const,
  tags: (id: RegistryProxyId, repository: string) =>
    [...registryProxyQueryKeys.base(), id, 'tags', repository] as const,
  tag: (id: RegistryProxyId, repository: string, tag: string) =>
    [...registryProxyQueryKeys.tags(id, repository), tag] as const,
};
