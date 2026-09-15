import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { updateRegistryProxy } from '../registry-proxy.service';
import { RegistryProxyId, RegistryProxyPayload } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useUpdateRegistryProxy(id: RegistryProxyId) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: RegistryProxyPayload) =>
      updateRegistryProxy(id, payload),
    ...withError('Failed updating registry proxy'),
    ...withInvalidate(queryClient, [
      registryProxyQueryKeys.list(),
      registryProxyQueryKeys.item(id),
    ]),
  });
}
