import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getRegistryProxy } from '../registry-proxy.service';
import { RegistryProxyId } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useRegistryProxy(id: RegistryProxyId) {
  return useQuery({
    queryKey: registryProxyQueryKeys.item(id),
    queryFn: () => getRegistryProxy(id),
    enabled: id !== undefined,
    ...withError('Failed loading registry proxy'),
  });
}
