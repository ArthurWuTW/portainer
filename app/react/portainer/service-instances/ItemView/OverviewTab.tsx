import { DetailsTable } from '@@/DetailsTable';
import { Widget } from '@@/Widget';

import { ServiceInstance, ServiceInstanceTargetTypes } from '../types';

interface Props {
  instance: ServiceInstance;
}

export function OverviewTab({ instance }: Props) {
  const targetLabel =
    instance.TargetType === ServiceInstanceTargetTypes.GROUP
      ? `Group #${instance.GroupId}`
      : `${instance.EnvironmentIds?.length ?? 0} environments`;

  return (
    <Widget>
      <Widget.Title title="Overview" />
      <Widget.Body>
        <DetailsTable
          dataCy="service-instance-overview"
          headers={['Property', 'Value']}
        >
          <DetailsTable.Row label="Name">{instance.Name}</DetailsTable.Row>
          <DetailsTable.Row label="Description">
            {instance.Description || '-'}
          </DetailsTable.Row>
          <DetailsTable.Row label="Target mode">
            {instance.TargetType === ServiceInstanceTargetTypes.GROUP
              ? 'Environment group'
              : 'Individual environments'}
          </DetailsTable.Row>
          <DetailsTable.Row label="Targets">{targetLabel}</DetailsTable.Row>
          <DetailsTable.Row label="Stack name">
            {instance.StackName}
          </DetailsTable.Row>
          <DetailsTable.Row label="Created by">
            {instance.CreatedBy}
          </DetailsTable.Row>
          <DetailsTable.Row label="Created at">
            {new Date(instance.CreatedAt * 1000).toLocaleString()}
          </DetailsTable.Row>
          <DetailsTable.Row label="Last updated">
            {new Date(instance.UpdatedAt * 1000).toLocaleString()}
          </DetailsTable.Row>
        </DetailsTable>
      </Widget.Body>
    </Widget>
  );
}
