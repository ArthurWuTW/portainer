import { useRouter } from '@uirouter/react';

import { notifySuccess } from '@/portainer/services/notifications';
import { useIdParam } from '@/react/hooks/useIdParam';

import { PageHeader } from '@@/PageHeader';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

import { useServiceInstance } from '../queries/useServiceInstance';
import { useUpdateServiceInstance } from '../queries/useUpdateServiceInstance';
import {
  ServiceInstanceForm,
  ServiceInstanceFormValues,
} from '../CreateView/ServiceInstanceForm';
import { ServiceInstanceTargetTypes } from '../types';

export function EditView() {
  const id = useIdParam('id');
  const router = useRouter();
  const instanceQuery = useServiceInstance(id);
  const updateMutation = useUpdateServiceInstance();

  const instance = instanceQuery.data;

  if (instanceQuery.isLoading || !instance) {
    return (
      <>
        <PageHeader
          breadcrumbs={[
            { label: 'Service Instances', link: 'portainer.service-instances' },
            { label: 'Edit' },
          ]}
        />
        <div className="mx-4">
          <div className="h-64 animate-pulse rounded bg-gray-1" />
        </div>
      </>
    );
  }

  const initialValues: ServiceInstanceFormValues = {
    name: instance.Name,
    description: instance.Description,
    targetMode:
      instance.TargetType === ServiceInstanceTargetTypes.GROUP
        ? 'group'
        : 'environments',
    groupId: instance.GroupId ?? null,
    environmentIds: instance.EnvironmentIds ?? [],
    composeFile: instance.ComposeFile,
  };

  return (
    <>
      <PageHeader
        title="Edit service instance"
        breadcrumbs={[
          { label: 'Service Instances', link: 'portainer.service-instances' },
          { label: instance.Name, link: 'portainer.service-instances.item' },
          { label: 'Edit' },
        ]}
      />
      <StickyFooter.Container>
        <div className="mx-4">
          <ServiceInstanceForm
            initialValues={initialValues}
            onSubmit={handleSubmit}
            submitLabel="Save"
            submitLoadingLabel="Saving..."
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

    await updateMutation.mutateAsync(
      { id, payload },
      {
        onSuccess: () => {
          notifySuccess('Success', 'Service instance successfully updated');
          router.stateService.go('portainer.service-instances.item', { id });
        },
      }
    );
  }
}
