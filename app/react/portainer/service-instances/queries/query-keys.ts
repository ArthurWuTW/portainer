import { ServiceInstanceId, ServiceInstanceOperationId } from '../types';

export const serviceInstanceQueryKeys = {
  base: () => ['service-instances'] as const,
  list: () => [...serviceInstanceQueryKeys.base()] as const,
  item: (id: ServiceInstanceId) =>
    [...serviceInstanceQueryKeys.base(), id] as const,
  targets: (id: ServiceInstanceId) =>
    [...serviceInstanceQueryKeys.base(), id, 'targets'] as const,
  operations: (id: ServiceInstanceId) =>
    [...serviceInstanceQueryKeys.base(), id, 'operations'] as const,
  operation: (id: ServiceInstanceOperationId) =>
    [...serviceInstanceQueryKeys.base(), 'operations', id] as const,
};
