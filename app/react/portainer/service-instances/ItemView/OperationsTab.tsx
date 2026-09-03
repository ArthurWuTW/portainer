import { Badge } from '@@/Badge';
import { DetailsTable } from '@@/DetailsTable';
import { Widget } from '@@/Widget';

import {
  ServiceInstance,
  ServiceInstanceOperationStatuses,
  ServiceInstanceOperationTypes,
  ServiceInstanceTargetStatuses,
} from '../types';
import { useServiceInstanceOperations } from '../queries/useServiceInstanceOperations';

interface Props {
  instance: ServiceInstance;
}

const operationTypeLabel: Record<number, string> = {
  [ServiceInstanceOperationTypes.DEPLOY]: 'Deploy',
  [ServiceInstanceOperationTypes.START]: 'Start',
  [ServiceInstanceOperationTypes.STOP]: 'Stop',
  [ServiceInstanceOperationTypes.REDEPLOY]: 'Redeploy',
  [ServiceInstanceOperationTypes.REFRESH]: 'Refresh',
  [ServiceInstanceOperationTypes.RESTART]: 'Restart',
};

const operationStatusBadge: Record<
  number,
  { type: 'success' | 'danger' | 'warn' | 'info' | 'muted'; label: string }
> = {
  [ServiceInstanceOperationStatuses.PENDING]: {
    type: 'muted',
    label: 'Pending',
  },
  [ServiceInstanceOperationStatuses.RUNNING]: {
    type: 'info',
    label: 'Running',
  },
  [ServiceInstanceOperationStatuses.SUCCESS]: {
    type: 'success',
    label: 'Success',
  },
  [ServiceInstanceOperationStatuses.PARTIAL_SUCCESS]: {
    type: 'warn',
    label: 'Partial success',
  },
  [ServiceInstanceOperationStatuses.FAILED]: {
    type: 'danger',
    label: 'Failed',
  },
  [ServiceInstanceOperationStatuses.CANCELLED]: {
    type: 'muted',
    label: 'Cancelled',
  },
};

export function OperationsTab({ instance }: Props) {
  const operationsQuery = useServiceInstanceOperations(instance.Id);

  if (operationsQuery.isLoading) {
    return (
      <Widget>
        <Widget.Title title="Operations" />
        <Widget.Body loading />
      </Widget>
    );
  }

  if (operationsQuery.isError) {
    return (
      <Widget>
        <Widget.Title title="Operations" />
        <Widget.Body>
          <p className="text-error">
            Failed loading operations:{' '}
            {operationsQuery.error instanceof Error
              ? operationsQuery.error.message
              : 'Unknown error'}
          </p>
        </Widget.Body>
      </Widget>
    );
  }

  const operations = operationsQuery.data ?? [];

  return (
    <Widget>
      <Widget.Title title="Operations" />
      <Widget.Body>
        <DetailsTable
          dataCy="service-instance-operations"
          headers={['ID', 'Type', 'Status', 'Started', 'Finished', 'Results']}
          emptyMessage="No operations"
        >
          {operations.map((op) => {
            const status = operationStatusBadge[op.Status] ?? {
              type: 'muted' as const,
              label: 'Unknown',
            };
            return (
              <tr key={op.Id} data-cy={`service-instance-operation-${op.Id}`}>
                <td>{op.Id}</td>
                <td>{operationTypeLabel[op.Type] ?? 'Unknown'}</td>
                <td>
                  <Badge type={status.type}>{status.label}</Badge>
                </td>
                <td>{new Date(op.StartedAt * 1000).toLocaleString()}</td>
                <td>
                  {op.FinishedAt
                    ? new Date(op.FinishedAt * 1000).toLocaleString()
                    : '-'}
                </td>
                <td>
                  {op.Results.map((result) => (
                    <div key={result.EnvironmentId}>
                      Env #{result.EnvironmentId}:{' '}
                      {result.Status === ServiceInstanceTargetStatuses.SUCCESS
                        ? 'success'
                        : result.Status === ServiceInstanceTargetStatuses.FAILED
                          ? `failed${result.Error ? ` (${result.Error})` : ''}`
                          : result.Status ===
                              ServiceInstanceTargetStatuses.SKIPPED
                            ? 'skipped'
                            : `status ${result.Status}`}
                    </div>
                  ))}
                </td>
              </tr>
            );
          })}
        </DetailsTable>
      </Widget.Body>
    </Widget>
  );
}
