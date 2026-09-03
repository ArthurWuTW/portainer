import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { cancelServiceInstanceScheduledBuild } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useCancelServiceInstanceScheduledBuild() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => cancelServiceInstanceScheduledBuild(id),
    ...withError('Failed cancelling scheduled build'),
    ...withInvalidate(queryClient, [
      serviceInstanceQueryKeys.scheduledBuilds(0),
    ]),
  });
}

export type CancelServiceInstanceScheduledBuildMutation = ReturnType<
  typeof useCancelServiceInstanceScheduledBuild
>;
