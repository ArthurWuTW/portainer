import { useRouter } from '@uirouter/react';

import { notifySuccess } from '@/portainer/services/notifications';
import { useIdParam } from '@/react/hooks/useIdParam';

import { PageHeader } from '@@/PageHeader';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

import { useRegistryProxy } from '../queries/useRegistryProxy';
import { useUpdateRegistryProxy } from '../queries/useUpdateRegistryProxy';
import {
  RegistryProxyForm,
  RegistryProxyFormValues,
} from '../CreateView/RegistryProxyForm';

export function EditView() {
  const id = useIdParam('id');
  const router = useRouter();
  const proxyQuery = useRegistryProxy(id);
  const updateMutation = useUpdateRegistryProxy(id);

  const proxy = proxyQuery.data;

  if (proxyQuery.isLoading || !proxy) {
    return (
      <>
        <PageHeader
          breadcrumbs={[
            { label: 'Registry Proxies', link: 'portainer.registry-proxy' },
            { label: 'Edit' },
          ]}
        />
        <div className="mx-4">
          <div className="h-64 animate-pulse rounded bg-gray-1" />
        </div>
      </>
    );
  }

  const initialValues: RegistryProxyFormValues = {
    name: proxy.Name,
    url: proxy.URL,
    tls: proxy.TLS,
    tlsSkipVerify: proxy.TLSSkipVerify,
    authentication: proxy.Authentication,
    username: proxy.Username ?? '',
    password: '',
  };

  return (
    <>
      <PageHeader
        title="Edit local registry"
        breadcrumbs={[
          { label: 'Registry Proxies', link: 'portainer.registry-proxy' },
          {
            label: proxy.Name,
            link: 'portainer.registry-proxy.item',
            linkParams: { id },
          },
          { label: 'Edit' },
        ]}
      />
      <StickyFooter.Container>
        <div className="mx-4">
          <RegistryProxyForm
            initialValues={initialValues}
            onSubmit={handleSubmit}
            submitLabel="Save changes"
            submitLoadingLabel="Saving..."
            isEditing
          />
        </div>
      </StickyFooter.Container>
    </>
  );

  async function handleSubmit(values: RegistryProxyFormValues) {
    const payload = {
      Name: values.name,
      URL: values.url,
      TLS: values.tls,
      TLSSkipVerify: values.tls ? values.tlsSkipVerify : false,
      Authentication: values.authentication,
      Username: values.authentication ? values.username : '',
      Password: values.authentication ? values.password : '',
    };

    await updateMutation.mutateAsync(payload, {
      onSuccess: () => {
        notifySuccess('Success', 'Local registry successfully updated');
        router.stateService.go('portainer.registry-proxy.item', { id });
      },
    });
  }
}
