import { Badge } from '@@/Badge';
import { DetailsTable } from '@@/DetailsTable';

import { ServiceInstanceTarget } from '../types';

interface Props {
  targets: ServiceInstanceTarget[];
  dataCy: string;
}

export function TargetsTable({ targets, dataCy }: Props) {
  return (
    <DetailsTable
      dataCy={dataCy}
      headers={['Environment', 'Stack', 'Status', 'Missing']}
      emptyMessage="No targets"
    >
      {targets.map((target) => (
        <tr
          key={target.EnvironmentId}
          data-cy={`${dataCy}-target-${target.EnvironmentId}`}
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
  );
}
