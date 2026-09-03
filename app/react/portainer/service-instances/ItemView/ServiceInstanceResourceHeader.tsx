import { Boxes, Play, RefreshCw, RotateCw, Square } from 'lucide-react';

import { Badge } from '@@/Badge';
import { Button } from '@@/buttons';
import { Icon } from '@@/Icon';
import { ResourceDetailHeader } from '@@/ResourceDetailHeader/ResourceDetailHeader';

import { ServiceInstance, ServiceInstanceStatuses } from '../types';
import {
  useDeployServiceInstance,
  useRedeployServiceInstance,
  useRefreshServiceInstance,
  useStartServiceInstance,
  useStopServiceInstance,
} from '../queries/useServiceInstanceLifecycle';

const statusBadgeType: Record<
  number,
  'success' | 'danger' | 'warn' | 'info' | 'muted'
> = {
  [ServiceInstanceStatuses.RUNNING]: 'success',
  [ServiceInstanceStatuses.STOPPED]: 'muted',
  [ServiceInstanceStatuses.DEPLOYING]: 'info',
  [ServiceInstanceStatuses.PARTIAL]: 'warn',
  [ServiceInstanceStatuses.FAILED]: 'danger',
  [ServiceInstanceStatuses.UNKNOWN]: 'muted',
};

const statusLabel: Record<number, string> = {
  [ServiceInstanceStatuses.RUNNING]: 'Running',
  [ServiceInstanceStatuses.STOPPED]: 'Stopped',
  [ServiceInstanceStatuses.DEPLOYING]: 'Deploying',
  [ServiceInstanceStatuses.PARTIAL]: 'Partial',
  [ServiceInstanceStatuses.FAILED]: 'Failed',
  [ServiceInstanceStatuses.UNKNOWN]: 'Unknown',
};

interface Props {
  instance: ServiceInstance;
}

export function ServiceInstanceResourceHeader({ instance }: Props) {
  const deployMutation = useDeployServiceInstance();
  const startMutation = useStartServiceInstance();
  const stopMutation = useStopServiceInstance();
  const redeployMutation = useRedeployServiceInstance();
  const refreshMutation = useRefreshServiceInstance();

  const isBusy =
    deployMutation.isLoading ||
    startMutation.isLoading ||
    stopMutation.isLoading ||
    redeployMutation.isLoading;

  return (
    <ResourceDetailHeader
      icon={<Icon icon={Boxes} size="xl" />}
      title={instance.Name}
      badge={
        <Badge type={statusBadgeType[instance.Status] ?? 'muted'}>
          {statusLabel[instance.Status] ?? 'Unknown'}
        </Badge>
      }
      description={instance.Description || undefined}
      actionBar={
        <div className="flex items-center gap-2">
          <Button
            color="primary"
            icon={Play}
            disabled={isBusy}
            data-cy="service-instance-deploy-button"
            onClick={() => deployMutation.mutate(instance.Id)}
          >
            Deploy
          </Button>
          <Button
            color="secondary"
            icon={Play}
            disabled={isBusy}
            data-cy="service-instance-start-button"
            onClick={() => startMutation.mutate(instance.Id)}
          >
            Start
          </Button>
          <Button
            color="secondary"
            icon={Square}
            disabled={isBusy}
            data-cy="service-instance-stop-button"
            onClick={() => stopMutation.mutate(instance.Id)}
          >
            Stop
          </Button>
          <Button
            color="secondary"
            icon={RotateCw}
            disabled={isBusy}
            data-cy="service-instance-redeploy-button"
            onClick={() => redeployMutation.mutate(instance.Id)}
          >
            Redeploy
          </Button>
          <Button
            color="secondary"
            icon={RefreshCw}
            disabled={refreshMutation.isLoading}
            data-cy="service-instance-refresh-button"
            onClick={() => refreshMutation.mutate(instance.Id)}
          >
            Refresh
          </Button>
        </div>
      }
    />
  );
}
