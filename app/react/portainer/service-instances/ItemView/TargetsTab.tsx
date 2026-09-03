import { Badge } from '@@/Badge';
import { DetailsTable } from '@@/DetailsTable';
import { Widget } from '@@/Widget';

import { ServiceInstance } from '../types';
import { useServiceInstanceTargets } from '../queries/useServiceInstanceTargets';

interface Props {
  instance: ServiceInstance;
}

export function TargetsTab({ instance }: Props) {
  const targetsQuery = useServiceInstanceTargets(instance.Id);

  if (targetsQuery.isLoading) {
    return (
      <Widget>
        <Widget.Title title="Targets" />
        <Widget.Body loading />
      </Widget>
    );
  }

  if (targetsQuery.isError) {
    return (
      <Widget>
        <Widget.Title title="Targets" />
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
      <Widget.Title title="Targets" />
      <Widget.Body>
        <DetailsTable
          dataCy="service-instance-targets"
          headers={['Environment', 'Stack', 'Status', 'Missing']}
          emptyMessage="No targets"
        >
          {targets.map((target) => (
            <tr
              key={target.EnvironmentId}
              data-cy={`service-instance-target-${target.EnvironmentId}`}
            >
              <td>{target.Environment?.Name ?? `#${target.EnvironmentId}`}</td>
              <td>{target.Stack?.Name ?? '-'}</td>
              <td>
                {target.Stack ? (
                  <Badge
                    type={
                      target.Stack.Status === 1
                        ? 'success'
                        : target.Stack.Status === 4
                          ? 'danger'
                          : 'muted'
                    }
                  >
                    {target.Stack.Status === 1
                      ? 'Running'
                      : target.Stack.Status === 4
                        ? 'Error'
                        : 'Stopped'}
                  </Badge>
                ) : (
                  '-'
                )}
              </td>
              <td>{target.Missing ? 'Missing' : '-'}</td>
            </tr>
          ))}
        </DetailsTable>
      </Widget.Body>
    </Widget>
  );
}
