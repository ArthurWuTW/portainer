import { useMutation, useQueryClient } from '@tanstack/react-query';

import { withError, withInvalidate } from '@/react-tools/react-query';

import {
  deployServiceInstance,
  startServiceInstance,
  stopServiceInstance,
  restartServiceInstance,
  redeployServiceInstance,
  refreshServiceInstance,
} from '../service-instance.service';

import { serviceInstanceQueryKeys } from './query-keys';

function useLifecycleMutation(
  mutationFn: (id: number) => Promise<unknown>,
  errorMessage: string
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
    ...withError(errorMessage),
    ...withInvalidate(queryClient, [
      serviceInstanceQueryKeys.list(),
      serviceInstanceQueryKeys.item(0),
      serviceInstanceQueryKeys.targets(0),
      serviceInstanceQueryKeys.operations(0),
    ]),
  });
}

export function useDeployServiceInstance() {
  return useLifecycleMutation(
    deployServiceInstance,
    'Failed deploying service instance'
  );
}

export function useStartServiceInstance() {
  return useLifecycleMutation(
    startServiceInstance,
    'Failed starting service instance'
  );
}

export function useStopServiceInstance() {
  return useLifecycleMutation(
    stopServiceInstance,
    'Failed stopping service instance'
  );
}

export function useRestartServiceInstance() {
  return useLifecycleMutation(
    restartServiceInstance,
    'Failed restarting service instance'
  );
}

export function useRedeployServiceInstance() {
  return useLifecycleMutation(
    redeployServiceInstance,
    'Failed redeploying service instance'
  );
}

export function useRefreshServiceInstance() {
  return useLifecycleMutation(
    refreshServiceInstance,
    'Failed refreshing service instance'
  );
}
