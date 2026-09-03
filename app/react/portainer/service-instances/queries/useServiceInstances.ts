import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getServiceInstances } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useServiceInstances() {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.list(),
    queryFn: getServiceInstances,
    ...withError('Failed loading service instances'),
  });
}
