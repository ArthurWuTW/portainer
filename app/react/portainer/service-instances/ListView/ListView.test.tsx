import { render, screen } from '@testing-library/react';
import { HttpResponse } from 'msw';

import { http, server } from '@/setup-tests/server';
import { withTestQueryProvider } from '@/react/test-utils/withTestQuery';
import { withTestRouter } from '@/react/test-utils/withRouter';
import { withUserProvider } from '@/react/test-utils/withUserProvider';
import { UserViewModel } from '@/portainer/models/user';

import { mockServiceInstance } from '../test-utils/mocks';

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
});

function renderComponent() {
  const user = new UserViewModel({ Username: 'user', Role: 1 });

  const Wrapped = withTestQueryProvider(
    withUserProvider(withTestRouter(ListView), user)
  );

  return render(<Wrapped />);
}
