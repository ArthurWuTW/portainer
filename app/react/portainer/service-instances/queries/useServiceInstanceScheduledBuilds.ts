import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { ServiceInstanceScheduledBuildStatuses } from '../types';
import { getServiceInstanceScheduledBuilds } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

const POLLING_INTERVAL = 3000;

export function useServiceInstanceScheduledBuilds(id?: number) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.scheduledBuilds(id ?? 0),
    queryFn: () => getServiceInstanceScheduledBuilds(id as number),
    enabled: id !== undefined,
    refetchInterval: (data) => {
      if (
        data?.some(
          (build) =>
            build.Status === ServiceInstanceScheduledBuildStatuses.PENDING ||
            build.Status === ServiceInstanceScheduledBuildStatuses.PULLING ||
            build.Status === ServiceInstanceScheduledBuildStatuses.IMAGE_READY
        )
      ) {
        return POLLING_INTERVAL;
      }
      return false;
    },
    ...withError('Failed loading scheduled builds'),
  });
}
