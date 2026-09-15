import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getRegistryTags } from '../registry-proxy.service';
import { RegistryProxyId } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useRegistryProxyTag(
  id: RegistryProxyId,
  repository: string | null,
  tag: string | null
) {
  return useQuery({
    queryKey: registryProxyQueryKeys.tags(id, repository ?? ''),
    queryFn: () => getRegistryTags(id, repository ?? ''),
    select: (data) => data.tags.find((item) => item.name === tag),
    enabled: !!repository && !!tag,
    ...withError('Failed loading image version'),
  });
}
