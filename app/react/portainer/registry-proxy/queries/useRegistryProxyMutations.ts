import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { addRegistryTag, deleteRegistryImage } from '../registry-proxy.service';
import { RegistryProxyId, RegistryTagPayload } from '../types';

import { registryProxyQueryKeys } from './query-keys';

export function useDeleteRegistryImage(
  id: RegistryProxyId,
  repository: string | null
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (reference: string) =>
      deleteRegistryImage(id, repository ?? '', reference),
    ...withError('Failed deleting image'),
    ...withInvalidate(queryClient, [
      registryProxyQueryKeys.catalog(id),
      registryProxyQueryKeys.tags(id, repository ?? ''),
    ]),
  });
}

export function useAddRegistryTag(id: RegistryProxyId) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: RegistryTagPayload) => addRegistryTag(id, payload),
    ...withError('Failed tagging image'),
    ...withInvalidate(queryClient, [registryProxyQueryKeys.base()]),
  });
}
