import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { deleteRegistryProxy } from '../registry-proxy.service';

import { registryProxyQueryKeys } from './query-keys';

export function useDeleteRegistryProxy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteRegistryProxy,
    ...withError('Failed deleting registry proxy'),
    ...withInvalidate(queryClient, [registryProxyQueryKeys.list()]),
  });
}
