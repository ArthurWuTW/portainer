import { useQuery } from '@tanstack/react-query';

import { withError } from '@/react-tools/react-query';

import { getServiceInstance } from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

export function useServiceInstance(id?: number) {
  return useQuery({
    queryKey: serviceInstanceQueryKeys.item(id ?? 0),
    queryFn: () => getServiceInstance(id as number),
    enabled: id !== undefined,
    ...withError('Failed loading service instance'),
  });
}
