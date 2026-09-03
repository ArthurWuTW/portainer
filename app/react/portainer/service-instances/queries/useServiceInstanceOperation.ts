import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { ServiceInstanceOperationStatuses } from '../types';
import { getServiceInstanceOperation } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

const POLLING_INTERVAL = 3000;

export function useServiceInstanceOperation(id?: number) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.operation(id ?? 0),
    queryFn: () => getServiceInstanceOperation(id as number),
    enabled: id !== undefined,
    refetchInterval: (data) => {
      if (
        data &&
        (data.Status === ServiceInstanceOperationStatuses.RUNNING ||
          data.Status === ServiceInstanceOperationStatuses.PENDING)
      ) {
        return POLLING_INTERVAL;
      }
      return false;
    },
    ...withError('Failed loading service instance operation'),
  });
}
