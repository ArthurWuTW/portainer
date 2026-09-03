import { useState } from 'react';
import { Play, RefreshCw, Square } from 'lucide-react';

import { Button } from '@@/buttons';
import { Widget } from '@@/Widget';
import { SwitchField } from '@@/form-components/SwitchField';

import { ServiceInstance } from '../types';
import { useServiceInstanceTargets } from '../queries/useServiceInstanceTargets';
import {
  useRestartServiceInstance,
  useStartServiceInstance,
  useStopServiceInstance,
} from '../queries/useServiceInstanceLifecycle';

import { TargetsTable } from './TargetsTable';

const MONITOR_REFRESH_INTERVAL_MS = 3000;

interface Props {
  instance: ServiceInstance;
}

export function MonitorTab({ instance }: Props) {
  const [autoRefresh, setAutoRefresh] = useState(true);

  const targetsQuery = useServiceInstanceTargets(instance.Id, {
    refetchInterval: autoRefresh ? MONITOR_REFRESH_INTERVAL_MS : false,
  });

  const startMutation = useStartServiceInstance();
  const stopMutation = useStopServiceInstance();
  const restartMutation = useRestartServiceInstance();

  const isBusy =
    startMutation.isLoading ||
    stopMutation.isLoading ||
    restartMutation.isLoading;

  if (targetsQuery.isLoading) {
    return (
      <Widget>
        <Widget.Title title="Monitor" />
        <Widget.Body loading />
      </Widget>
    );
  }

  if (targetsQuery.isError) {
    return (
      <Widget>
        <Widget.Title title="Monitor" />
        <Widget.Body>
          <p className="text-error">
            Failed loading targets:{' '}
            {targetsQuery.error instanceof Error
              ? targetsQuery.error.message
              : 'Unknown error'}
          </p>
        </Widget.Body>
      </Widget>
    );
  }

  const targets = targetsQuery.data ?? [];

  return (
    <Widget>
      <Widget.Title
        title="Monitor"
        subtitle={
          autoRefresh
            ? `Auto-refreshes every ${MONITOR_REFRESH_INTERVAL_MS / 1000} seconds`
            : 'Auto-refresh disabled'
        }
      >
        <div className="flex items-center gap-2">
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
            icon={RefreshCw}
            disabled={isBusy}
            data-cy="service-instance-restart-button"
            onClick={() => restartMutation.mutate(instance.Id)}
          >
            Restart
          </Button>
          <SwitchField
            label="Auto-refresh"
            name="monitorAutoRefresh"
            checked={autoRefresh}
            onChange={setAutoRefresh}
            data-cy="service-instance-monitor-auto-refresh"
          />
        </div>
      </Widget.Title>
      <Widget.Body>
        <TargetsTable targets={targets} dataCy="service-instance-monitor" />
      </Widget.Body>
    </Widget>
  );
}
