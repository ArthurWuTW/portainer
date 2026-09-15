import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getRegistryProxies } from '../registry-proxy.service';

import { registryProxyQueryKeys } from './query-keys';

export function useRegistryProxies() {
  return useQuery({
    queryKey: registryProxyQueryKeys.list(),
    queryFn: getRegistryProxies,
    ...withError('Failed loading registry proxies'),
  });
}
