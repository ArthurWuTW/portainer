import { render, screen } from '@testing-library/react';
import { HttpResponse } from 'msw';

import { http, server } from '@/setup-tests/server';
import { withTestQueryProvider } from '@/react/test-utils/withTestQuery';
import { withTestRouter } from '@/react/test-utils/withRouter';
import { withUserProvider } from '@/react/test-utils/withUserProvider';
import { UserViewModel } from '@/portainer/models/user';
import {
  createMockEnvironment,
  createMockEnvironmentGroup,
} from '@/react-tools/test-mocks';

import { mockServiceInstance } from '../test-utils/mocks';
import { ServiceInstanceTargetTypes } from '../types';

import { ListView } from './ListView';

describe('Service Instances ListView', () => {
  it('renders the list of service instances', async () => {
    server.use(
      http.get('/api/service-instances', () =>
        HttpResponse.json([mockServiceInstance])
      )
    );

    renderComponent();

    expect(await screen.findByText('production-web')).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('si-1-production-web')).toBeInTheDocument();
  });

  it('renders the empty state when no service instances exist', async () => {
    server.use(http.get('/api/service-instances', () => HttpResponse.json([])));

    renderComponent();

    expect(await screen.findByText('No service instances')).toBeInTheDocument();
  });

  it('displays target environment and group names in the Target column', async () => {
    server.use(
      http.get('/api/service-instances', () =>
        HttpResponse.json([
          mockServiceInstance,
          {
            ...mockServiceInstance,
            Id: 2,
            Name: 'env-targets',
            TargetType: ServiceInstanceTargetTypes.ENVIRONMENTS,
            GroupId: undefined,
            EnvironmentIds: [1, 2],
          },
        ])
      ),
      http.get('/api/endpoints', () =>
        HttpResponse.json([
          createMockEnvironment({ Id: 1, Name: 'prod-a' }),
          createMockEnvironment({ Id: 2, Name: 'prod-b' }),
        ])
      ),
      http.get('/api/endpoint_groups', () =>
        HttpResponse.json([
          createMockEnvironmentGroup({ Id: 1, Name: 'prod-group' }),
        ])
      )
    );

    renderComponent();

    expect(await screen.findByText('prod-group')).toBeInTheDocument();
    expect(screen.getByText('prod-a, prod-b')).toBeInTheDocument();
    expect(screen.queryByText(/Group #/)).not.toBeInTheDocument();
    expect(screen.queryByText(/environments$/)).not.toBeInTheDocument();
  });
});

function renderComponent() {
  const user = new UserViewModel({ Username: 'user', Role: 1 });

  const Wrapped = withTestQueryProvider(
    withUserProvider(withTestRouter(ListView), user)
  );

  return render(<Wrapped />);
}
