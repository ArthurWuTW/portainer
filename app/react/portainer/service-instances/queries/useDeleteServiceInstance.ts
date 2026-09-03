import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { deleteServiceInstance } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useDeleteServiceInstance() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteServiceInstance,
    ...withError('Failed deleting service instance'),
    ...withInvalidate(queryClient, [serviceInstanceQueryKeys.list()]),
  });
}

export type DeleteServiceInstanceMutation = ReturnType<
  typeof useDeleteServiceInstance
>;
