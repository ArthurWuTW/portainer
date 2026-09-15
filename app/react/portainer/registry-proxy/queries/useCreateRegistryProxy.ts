import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { createRegistryProxy } from '../registry-proxy.service';

import { registryProxyQueryKeys } from './query-keys';

export function useCreateRegistryProxy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createRegistryProxy,
    ...withError('Failed creating registry proxy'),
    ...withInvalidate(queryClient, [registryProxyQueryKeys.list()]),
  });
}
