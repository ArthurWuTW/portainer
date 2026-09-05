import { useEffect, useRef, useState } from 'react';
import { CalendarClock, Play, X } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';

import { notifySuccess } from '@/portainer/services/notifications';
import { isoDateFromTimestamp } from '@/portainer/filters/filters';

import { Badge } from '@@/Badge';
import { Button } from '@@/buttons';
import { CodeEditor } from '@@/CodeEditor';
import { DateTimeField } from '@@/DateTimeField';
import { DetailsTable } from '@@/DetailsTable';
import { Widget } from '@@/Widget';

import {
  ServiceInstance,
  ServiceInstanceScheduledBuild,
  ServiceInstanceScheduledBuildStatuses,
  ServiceInstanceScheduledBuildTargetStatuses,
} from '../types';
import { useScheduleServiceInstanceBuild } from '../queries/useScheduleServiceInstanceBuild';
import { useCancelServiceInstanceScheduledBuild } from '../queries/useCancelServiceInstanceScheduledBuild';
import { useServiceInstanceScheduledBuilds } from '../queries/useServiceInstanceScheduledBuilds';
import { serviceInstanceQueryKeys } from '../queries/query-keys';

interface Props {
  instance: ServiceInstance;
}

const buildStatusBadge: Record<
  number,
  { type: 'success' | 'danger' | 'warn' | 'info' | 'muted'; label: string }
> = {
  [ServiceInstanceScheduledBuildStatuses.PENDING]: {
    type: 'muted',
    label: 'Pending',
  },
  [ServiceInstanceScheduledBuildStatuses.PULLING]: {
    type: 'info',
    label: 'Pulling images',
  },
  [ServiceInstanceScheduledBuildStatuses.IMAGE_READY]: {
    type: 'info',
    label: 'Image ready',
  },
  [ServiceInstanceScheduledBuildStatuses.DEPLOYED]: {
    type: 'success',
    label: 'Deployed',
  },
  [ServiceInstanceScheduledBuildStatuses.FAILED]: {
    type: 'danger',
    label: 'Failed',
  },
  [ServiceInstanceScheduledBuildStatuses.CANCELLED]: {
    type: 'muted',
    label: 'Cancelled',
  },
};

const targetStatusBadge: Record<
  number,
  { type: 'success' | 'danger' | 'warn' | 'info' | 'muted'; label: string }
> = {
  [ServiceInstanceScheduledBuildTargetStatuses.PENDING]: {
    type: 'muted',
    label: 'Pending',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.PULLING]: {
    type: 'info',
    label: 'Pulling',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.IMAGE_READY]: {
    type: 'info',
    label: 'Image ready',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.DEPLOYED]: {
    type: 'success',
    label: 'Deployed',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.FAILED]: {
    type: 'danger',
    label: 'Failed',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.SKIPPED]: {
    type: 'muted',
    label: 'Skipped',
  },
  [ServiceInstanceScheduledBuildTargetStatuses.CANCELLED]: {
    type: 'muted',
    label: 'Cancelled',
  },
};

function isCancellable(build: ServiceInstanceScheduledBuild) {
  return (
    build.Status === ServiceInstanceScheduledBuildStatuses.PENDING ||
    build.Status === ServiceInstanceScheduledBuildStatuses.PULLING ||
    build.Status === ServiceInstanceScheduledBuildStatuses.IMAGE_READY
  );
}

export function DeployTab({ instance }: Props) {
  const [compose, setCompose] = useState(instance.ComposeFile);
  const [deployAt, setDeployAt] = useState<Date | null>(null);

  const queryClient = useQueryClient();
  const scheduleMutation = useScheduleServiceInstanceBuild();
  const cancelMutation = useCancelServiceInstanceScheduledBuild();
  const scheduledBuildsQuery = useServiceInstanceScheduledBuilds(instance.Id);

  const builds = (scheduledBuildsQuery.data ?? []).slice().sort((a, b) => b.Id - a.Id);
  const hasActiveBuild = builds.some(isCancellable);

  // When a scheduled build finishes, refresh the instance so the Compose tab
  // shows the latest compose file.
  const prevHasActiveBuild = useRef(hasActiveBuild);
  useEffect(() => {
    if (prevHasActiveBuild.current && !hasActiveBuild) {
      queryClient.invalidateQueries(
        serviceInstanceQueryKeys.item(instance.Id)
      );
    }
    prevHasActiveBuild.current = hasActiveBuild;
  }, [hasActiveBuild, instance.Id, queryClient]);

  const isBusy = scheduleMutation.isLoading;

  async function handleDeploy() {
    await scheduleMutation.mutateAsync({
      id: instance.Id,
      payload: {
        ComposeFile: compose,
        DeployAt: Math.floor(Date.now() / 1000),
      },
    });
    notifySuccess('Success', 'Deployment started');
  }

  async function handleSchedule() {
    if (!deployAt) {
      return;
    }
    await scheduleMutation.mutateAsync({
      id: instance.Id,
      payload: {
        ComposeFile: compose,
        DeployAt: Math.floor(deployAt.getTime() / 1000),
      },
    });
    notifySuccess('Success', 'Build scheduled');
  }

  return (
    <div className="space-y-4">
      <Widget>
        <Widget.Title title="Compose" />
        <Widget.Body>
          <CodeEditor
            id="service-instance-deploy-compose"
            value={compose}
            onChange={setCompose}
            height="400px"
            data-cy="service-instance-deploy-compose-editor"
          />
          <div className="mt-4 flex flex-wrap items-end gap-4">
            <Button
              color="primary"
              icon={Play}
              disabled={isBusy}
              data-cy="service-instance-deploy-now-button"
              onClick={handleDeploy}
            >
              Deploy
            </Button>
            <div className="flex flex-col gap-1">
              <DateTimeField
                label="Deploy at"
                name="deploy-at"
                value={deployAt}
                onChange={setDeployAt}
                minDate={new Date()}
                data-cy="service-instance-schedule-deploy-at"
              />
              <Button
                color="secondary"
                icon={CalendarClock}
                disabled={isBusy || !deployAt}
                data-cy="service-instance-schedule-build-button"
                onClick={handleSchedule}
              >
                Schedule build
              </Button>
            </div>
          </div>
        </Widget.Body>
      </Widget>

      <Widget>
        <Widget.Title title="Scheduled builds" />
        <Widget.Body>
          <DetailsTable
            dataCy="service-instance-scheduled-builds"
            headers={['ID', 'Deploy at', 'Status', 'Targets', 'Error', 'Actions']}
            emptyMessage="No scheduled builds"
          >
            {builds.map((build) => {
              const status = buildStatusBadge[build.Status] ?? {
                type: 'muted' as const,
                label: 'Unknown',
              };
              return (
                <tr
                  key={build.Id}
                  data-cy={`service-instance-scheduled-build-${build.Id}`}
                >
                  <td>{build.Id}</td>
                  <td>{isoDateFromTimestamp(build.DeployAt)}</td>
                  <td>
                    <Badge type={status.type}>{status.label}</Badge>
                  </td>
                  <td>
                    {(build.Results ?? []).map((result) => {
                      const targetStatus =
                        targetStatusBadge[result.Status] ?? {
                          type: 'muted' as const,
                          label: 'Unknown',
                        };
                      return (
                        <div
                          key={result.EnvironmentId}
                          data-cy={`service-instance-scheduled-build-target-${build.Id}-${result.EnvironmentId}`}
                        >
                          Env #{result.EnvironmentId}:{' '}
                          <Badge type={targetStatus.type}>
                            {targetStatus.label}
                          </Badge>
                          {result.Error ? ` (${result.Error})` : ''}
                        </div>
                      );
                    })}
                  </td>
                  <td>{build.Error || '-'}</td>
                  <td>
                    {isCancellable(build) && (
                      <Button
                        color="dangerlight"
                        size="xsmall"
                        icon={X}
                        disabled={cancelMutation.isLoading}
                        data-cy={`service-instance-scheduled-build-cancel-${build.Id}`}
                        onClick={() => cancelMutation.mutate(build.Id)}
                      >
                        Cancel
                      </Button>
                    )}
                  </td>
                </tr>
              );
            })}
          </DetailsTable>
        </Widget.Body>
      </Widget>
    </div>
  );
}
