import { useRouter } from '@uirouter/react';

import { notifySuccess } from '@/portainer/services/notifications';

import { PageHeader } from '@@/PageHeader';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

import { useCreateServiceInstance } from '../queries/useCreateServiceInstance';
import { ServiceInstanceTargetTypes } from '../types';

import {
  ServiceInstanceForm,
  ServiceInstanceFormValues,
} from './ServiceInstanceForm';

export function CreateView() {
  const router = useRouter();
  const createMutation = useCreateServiceInstance();

  const initialValues: ServiceInstanceFormValues = {
    name: '',
    description: '',
    targetMode: 'group',
    groupId: null,
    environmentIds: [],
    composeFile: '',
  };

  return (
    <>
      <PageHeader
        title="Create service instance"
        breadcrumbs={[
          { label: 'Service Instances', link: 'portainer.service-instances' },
          { label: 'Create' },
        ]}
      />
      <StickyFooter.Container>
        <div className="mx-4">
          <ServiceInstanceForm
            initialValues={initialValues}
            onSubmit={handleSubmit}
            submitLabel="Create"
            submitLoadingLabel="Creating..."
          />
        </div>
      </StickyFooter.Container>
    </>
  );

  async function handleSubmit(values: ServiceInstanceFormValues) {
    const payload = {
      Name: values.name,
      Description: values.description,
      TargetType:
        values.targetMode === 'group'
          ? ServiceInstanceTargetTypes.GROUP
          : ServiceInstanceTargetTypes.ENVIRONMENTS,
      GroupId:
        values.targetMode === 'group'
          ? (values.groupId ?? undefined)
          : undefined,
      EnvironmentIds:
        values.targetMode === 'environments'
          ? values.environmentIds
          : undefined,
      ComposeFile: values.composeFile,
    };

    await createMutation.mutateAsync(payload, {
      onSuccess: () => {
        notifySuccess('Success', 'Service instance successfully created');
        router.stateService.go('portainer.service-instances');
      },
    });
  }
}
