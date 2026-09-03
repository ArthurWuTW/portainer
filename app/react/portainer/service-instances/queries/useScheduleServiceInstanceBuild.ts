import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import { ScheduleServiceInstanceBuildPayload } from '../types';
import { scheduleServiceInstanceBuild } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useScheduleServiceInstanceBuild() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: number;
      payload: ScheduleServiceInstanceBuildPayload;
    }) => scheduleServiceInstanceBuild(id, payload),
    ...withError('Failed scheduling build'),
    ...withInvalidate(queryClient, [
      serviceInstanceQueryKeys.list(),
      serviceInstanceQueryKeys.item(0),
      serviceInstanceQueryKeys.scheduledBuilds(0),
    ]),
  });
}

export type ScheduleServiceInstanceBuildMutation = ReturnType<
  typeof useScheduleServiceInstanceBuild
>;
