import { Widget } from '@@/Widget';

import { ServiceInstance } from '../types';
import { useServiceInstanceTargets } from '../queries/useServiceInstanceTargets';

import { TargetsTable } from './TargetsTable';

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
        <TargetsTable targets={targets} dataCy="service-instance-targets" />
      </Widget.Body>
    </Widget>
  );
}
