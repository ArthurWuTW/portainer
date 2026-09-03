import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { createServiceInstance } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useCreateServiceInstance() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createServiceInstance,
    ...withError('Failed creating service instance'),
    ...withInvalidate(queryClient, [serviceInstanceQueryKeys.list()]),
  });
}

export type CreateServiceInstanceMutation = ReturnType<
  typeof useCreateServiceInstance
>;
