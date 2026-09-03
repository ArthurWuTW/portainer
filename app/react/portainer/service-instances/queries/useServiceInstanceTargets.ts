import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getServiceInstanceTargets } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useServiceInstanceTargets(id?: number) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.targets(id ?? 0),
    queryFn: () => getServiceInstanceTargets(id as number),
    enabled: id !== undefined,
    ...withError('Failed loading service instance targets'),
  });
}
