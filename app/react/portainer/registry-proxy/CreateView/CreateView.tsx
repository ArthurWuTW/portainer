import { useRouter } from '@uirouter/react';

import { notifySuccess } from '@/portainer/services/notifications';

import { PageHeader } from '@@/PageHeader';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

import { useCreateRegistryProxy } from '../queries/useCreateRegistryProxy';

import {
  RegistryProxyForm,
  RegistryProxyFormValues,
} from './RegistryProxyForm';

export function CreateView() {
  const router = useRouter();
  const createMutation = useCreateRegistryProxy();

  const initialValues: RegistryProxyFormValues = {
    name: '',
    url: '',
    tls: false,
    tlsSkipVerify: false,
    authentication: false,
    username: '',
    password: '',
  };

  return (
    <>
      <PageHeader
        title="Register local registry"
        breadcrumbs={[
          {
            label: 'Registry Proxies',
            link: 'portainer.registry-proxy',
          },
          { label: 'Register' },
        ]}
      />
      <StickyFooter.Container>
        <div className="mx-4">
          <RegistryProxyForm
            initialValues={initialValues}
            onSubmit={handleSubmit}
            submitLabel="Register registry"
            submitLoadingLabel="Registering..."
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

    await createMutation.mutateAsync(payload, {
      onSuccess: () => {
        notifySuccess('Success', 'Local registry successfully registered');
        router.stateService.go('portainer.registry-proxy');
      },
    });
  }
}
