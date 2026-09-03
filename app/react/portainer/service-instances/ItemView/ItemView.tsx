import { useState } from 'react';

import { useIdParam } from '@/react/hooks/useIdParam';

import { Alert } from '@@/Alert';
import { PageHeader } from '@@/PageHeader';
import { ResourceDetailHeaderSkeleton } from '@@/ResourceDetailHeader/ResourceDetailHeaderSkeleton';
import { NavTabs } from '@@/NavTabs';

import { useServiceInstance } from '../queries/useServiceInstance';

import { ServiceInstanceResourceHeader } from './ServiceInstanceResourceHeader';
import { OverviewTab } from './OverviewTab';
import { TargetsTab } from './TargetsTab';
import { MonitorTab } from './MonitorTab';
import { ComposeTab } from './ComposeTab';
import { DeployTab } from './DeployTab';
import { OperationsTab } from './OperationsTab';

const breadcrumbs = [
  { label: 'Service Instances', link: 'portainer.service-instances' },
  'Service Instance',
];

type TabId =
  | 'overview'
  | 'targets'
  | 'monitor'
  | 'compose'
  | 'deploy'
  | 'operations';

export function ItemView() {
  const id = useIdParam('id');
  const instanceQuery = useServiceInstance(id);
  const instance = instanceQuery.data;

  const [selectedTab, setSelectedTab] = useState<TabId>('overview');

  if (instanceQuery.isLoading) {
    return (
      <>
        <PageHeader breadcrumbs={breadcrumbs} />
        <div className="mx-4 mb-4 space-y-4">
          <ResourceDetailHeaderSkeleton statBlockCount={1} />
        </div>
      </>
    );
  }

  if (!instance || instanceQuery.isError) {
    const error = instanceQuery.error;

    return (
      <>
        <PageHeader breadcrumbs={breadcrumbs} />
        <div className="mx-4 mb-4 space-y-4">
          <Alert color="error">
            Failed loading service instance:{' '}
            {error instanceof Error ? error.message : 'Unknown error'}
          </Alert>
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader
        breadcrumbs={[
          { label: 'Service Instances', link: 'portainer.service-instances' },
          instance.Name,
        ]}
        reload
      />
      <div className="mx-4 space-y-4 pb-4">
        <ServiceInstanceResourceHeader instance={instance} />
        <NavTabs
          options={[
            { id: 'overview', label: 'Overview' },
            { id: 'targets', label: 'Targets' },
            { id: 'monitor', label: 'Monitor' },
            { id: 'compose', label: 'Compose' },
            { id: 'deploy', label: 'Deploy' },
            { id: 'operations', label: 'Operations' },
          ]}
          selectedId={selectedTab}
          onSelect={(id) => setSelectedTab(id as TabId)}
        />
        {selectedTab === 'overview' && <OverviewTab instance={instance} />}
        {selectedTab === 'targets' && <TargetsTab instance={instance} />}
        {selectedTab === 'monitor' && <MonitorTab instance={instance} />}
        {selectedTab === 'compose' && <ComposeTab instance={instance} />}
        {selectedTab === 'deploy' && <DeployTab instance={instance} />}
        {selectedTab === 'operations' && <OperationsTab instance={instance} />}
      </div>
    </>
  );
}
