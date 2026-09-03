import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { ReactNode } from 'react';
import { HttpResponse } from 'msw';

import { http, server } from '@/setup-tests/server';
import { withTestQueryProvider } from '@/react/test-utils/withTestQuery';
import { withTestRouter } from '@/react/test-utils/withRouter';
import { withUserProvider } from '@/react/test-utils/withUserProvider';
import { UserViewModel } from '@/portainer/models/user';

import {
  mockServiceInstance,
  mockServiceInstanceOperation,
  mockServiceInstanceTargets,
} from '../test-utils/mocks';
import {
  ServiceInstanceOperationStatuses,
  ServiceInstanceTargetStatuses,
} from '../types';

import { ItemView } from './ItemView';

const useCurrentStateAndParams = vi.fn(() => ({
  params: { id: '1' },
}));
const go = vi.fn();

vi.mock('@uirouter/react', async (importOriginal: () => Promise<object>) => ({
  ...(await importOriginal()),
  useCurrentStateAndParams: () => useCurrentStateAndParams(),
  useRouter: () => ({ stateService: { go } }),
}));

// Avoid ui-router relative/unregistered state resolution in tests
vi.mock('@@/Link', () => ({
  Link: ({
    children,
    'data-cy': dataCy,
    className,
  }: {
    children: ReactNode;
    'data-cy'?: string;
    className?: string;
  }) => (
    <a data-cy={dataCy} className={className} href="/">
      {children}
    </a>
  ),
}));

describe('Service Instance ItemView', () => {
  it('renders the instance name and status badge', async () => {
    renderComponent();

    expect(
      await screen.findByRole('heading', { name: 'production-web' })
    ).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
  });

  it('renders the tab navigation', async () => {
    renderComponent();

    await screen.findByRole('heading', { name: 'production-web' });

    expect(
      screen.getByRole('button', { name: 'Overview' })
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Targets' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Monitor' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Compose' })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Operations' })
    ).toBeInTheDocument();
  });

  it('renders the overview details', async () => {
    renderComponent();

    await screen.findByRole('heading', { name: 'production-web' });

    expect(
      screen.getAllByText('production web service').length
    ).toBeGreaterThan(0);
    expect(screen.getByText('Environment group')).toBeInTheDocument();
    expect(screen.getAllByText('si-1-production-web').length).toBeGreaterThan(
      0
    );
  });

  it('displays per-target results including failures in the operations tab', async () => {
    const failedOperation = {
      ...mockServiceInstanceOperation,
      Id: 2,
      Status: ServiceInstanceOperationStatuses.PARTIAL_SUCCESS,
      Results: [
        { EnvironmentId: 1, Status: ServiceInstanceTargetStatuses.SUCCESS },
        {
          EnvironmentId: 2,
          Status: ServiceInstanceTargetStatuses.FAILED,
          Error: 'image pull failed',
        },
        { EnvironmentId: 3, Status: ServiceInstanceTargetStatuses.SKIPPED },
      ],
    };

    renderComponent({ operations: [failedOperation] });

    await screen.findByRole('heading', { name: 'production-web' });
    await userEvent.click(screen.getByRole('button', { name: 'Operations' }));

    expect(await screen.findByText('Partial success')).toBeInTheDocument();
    expect(screen.getByText(/Env #1: success/)).toBeInTheDocument();
    expect(
      screen.getByText(/Env #2: failed \(image pull failed\)/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Env #3: skipped/)).toBeInTheDocument();
  });

  it('displays the auto-refreshing monitor of target environments', async () => {
    renderComponent();

    await screen.findByRole('heading', { name: 'production-web' });
    await userEvent.click(screen.getByRole('button', { name: 'Monitor' }));

    expect(
      await screen.findByRole('heading', { name: 'Monitor' })
    ).toBeInTheDocument();
    expect(
      await screen.findByText('Auto-refreshes every 0.5 seconds')
    ).toBeInTheDocument();
    expect(screen.getByText('prod-a')).toBeInTheDocument();
    expect(screen.getByText('prod-b')).toBeInTheDocument();
    expect(
      screen.getAllByText('si-1-production-web').length
    ).toBeGreaterThanOrEqual(1);
  });

  it('toggles auto-refresh via the monitor switch', async () => {
    renderComponent();

    await screen.findByRole('heading', { name: 'production-web' });
    await userEvent.click(screen.getByRole('button', { name: 'Monitor' }));

    expect(
      await screen.findByText('Auto-refreshes every 0.5 seconds')
    ).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole('checkbox', { name: 'Auto-refresh' })
    );

    expect(screen.getByText('Auto-refresh disabled')).toBeInTheDocument();
  });

  it('displays missing targets in the targets tab', async () => {
    renderComponent({
      targets: [
        ...mockServiceInstanceTargets,
        { EnvironmentId: 99, Missing: true },
      ],
    });

    await screen.findByRole('heading', { name: 'production-web' });
    await userEvent.click(screen.getByRole('button', { name: 'Targets' }));

    expect(await screen.findByText('prod-a')).toBeInTheDocument();
    expect(screen.getByText('prod-b')).toBeInTheDocument();
    // "Missing" appears in the column header and in the missing-target cell
    expect(screen.getAllByText('Missing').length).toBeGreaterThanOrEqual(2);
  });
});

function renderComponent(
  overrides: {
    operations?: (typeof mockServiceInstanceOperation)[];
    targets?: typeof mockServiceInstanceTargets;
  } = {}
) {
  go.mockClear();
  useCurrentStateAndParams.mockReturnValue({
    params: { id: '1' },
  });

  server.use(
    http.get('/api/service-instances/1', () =>
      HttpResponse.json(mockServiceInstance)
    ),
    http.get('/api/service-instances/1/targets', () =>
      HttpResponse.json(overrides.targets ?? mockServiceInstanceTargets)
    ),
    http.get('/api/service-instances/1/operations', () =>
      HttpResponse.json(overrides.operations ?? [mockServiceInstanceOperation])
    )
  );

  const user = new UserViewModel({ Username: 'user', Role: 1 });

  const Wrapped = withTestQueryProvider(
    withUserProvider(withTestRouter(ItemView), user)
  );

  return render(<Wrapped />);
}
