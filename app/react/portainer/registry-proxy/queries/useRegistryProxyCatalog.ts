import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getRegistryCatalog } from '../registry-proxy.service';
import { RegistryProxyId } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useRegistryProxyCatalog(id: RegistryProxyId) {
  return useQuery({
    queryKey: registryProxyQueryKeys.catalog(id),
    queryFn: () => getRegistryCatalog(id),
    ...withError('Failed loading registry image list'),
  });
}
