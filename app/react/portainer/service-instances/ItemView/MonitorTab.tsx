import { useState } from 'react';

import { Widget } from '@@/Widget';
import { SwitchField } from '@@/form-components/SwitchField';

import { ServiceInstance } from '../types';
import { useServiceInstanceTargets } from '../queries/useServiceInstanceTargets';

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
        <SwitchField
          label="Auto-refresh"
          name="monitorAutoRefresh"
          checked={autoRefresh}
          onChange={setAutoRefresh}
          data-cy="service-instance-monitor-auto-refresh"
        />
      </Widget.Title>
      <Widget.Body>
        <TargetsTable targets={targets} dataCy="service-instance-monitor" />
      </Widget.Body>
    </Widget>
  );
}
