import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { UpdateServiceInstancePayload } from '../types';
import { updateServiceInstance } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useUpdateServiceInstance() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: number;
      payload: UpdateServiceInstancePayload;
    }) => updateServiceInstance(id, payload),
    ...withError('Failed updating service instance'),
    ...withInvalidate(queryClient, [
      serviceInstanceQueryKeys.list(),
      serviceInstanceQueryKeys.item(0),
    ]),
  });
}

export type UpdateServiceInstanceMutation = ReturnType<
  typeof useUpdateServiceInstance
>;
