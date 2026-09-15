import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getRegistryTags } from '../registry-proxy.service';
import { RegistryProxyId } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useRegistryProxyTags(
  id: RegistryProxyId,
  repository: string | null
) {
  return useQuery({
    queryKey: registryProxyQueryKeys.tags(id, repository ?? ''),
    queryFn: () => getRegistryTags(id, repository ?? ''),
    enabled: !!repository,
    ...withError('Failed loading image versions'),
  });
}
