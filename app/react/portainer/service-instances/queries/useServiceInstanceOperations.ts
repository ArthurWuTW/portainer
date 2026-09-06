import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';
import { PaginationQueryParams } from '@/react/common/api/pagination.types';

import {
  ServiceInstanceOperation,
  ServiceInstanceOperationStatuses,
} from '../types';
import { getServiceInstanceOperations } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

const POLLING_INTERVAL = 3000;

function isOperationActive(
  operations?: ServiceInstanceOperation[] | null
) {
  return (operations ?? []).some(
    (op) =>
      op.Status === ServiceInstanceOperationStatuses.PENDING ||
      op.Status === ServiceInstanceOperationStatuses.RUNNING
  );
}

export function useServiceInstanceOperations(
  id?: number,
  params?: PaginationQueryParams
) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.operations(id ?? 0, params),
    queryFn: () => getServiceInstanceOperations(id as number, params),
    enabled: id !== undefined,
    refetchInterval: (data) =>
      isOperationActive(data?.data) ? POLLING_INTERVAL : false,
    ...withError('Failed loading service instance operations'),
  });
}
