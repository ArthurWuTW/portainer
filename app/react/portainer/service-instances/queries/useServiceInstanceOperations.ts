import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import {
  ServiceInstanceOperation,
  ServiceInstanceOperationStatuses,
} from '../types';
import { getServiceInstanceOperations } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

const POLLING_INTERVAL = 3000;

function isOperationActive(operations?: ServiceInstanceOperation[]) {
  return (operations ?? []).some(
    (op) =>
      op.Status === ServiceInstanceOperationStatuses.PENDING ||
      op.Status === ServiceInstanceOperationStatuses.RUNNING
  );
}

export function useServiceInstanceOperations(id?: number) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.operations(id ?? 0),
    queryFn: () => getServiceInstanceOperations(id as number),
    enabled: id !== undefined,
    refetchInterval: (data) =>
      isOperationActive(data) ? POLLING_INTERVAL : false,
    ...withError('Failed loading service instance operations'),
  });
}
