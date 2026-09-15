import { DetailsTable } from '@@/DetailsTable';

import { RegistryProxy } from '../types';

interface Props {
  proxy?: RegistryProxy;
  proxyPath: string;
}

export function ProxyDetailsPanel({ proxy, proxyPath }: Props) {
  return (
    <div className="flex flex-col gap-4">
      <DetailsTable dataCy="registry-proxy-details">
        <DetailsTable.Row label="Name">{proxy?.Name ?? '-'}</DetailsTable.Row>
        <DetailsTable.Row label="Local registry URL">
          {proxy?.URL ?? '-'}
        </DetailsTable.Row>
        <DetailsTable.Row label="TLS">
          {proxy?.TLS ? 'Enabled' : 'Disabled'}
        </DetailsTable.Row>
        <DetailsTable.Row label="Registry authentication">
          {proxy?.Authentication ? `Enabled (${proxy.Username})` : 'Disabled'}
        </DetailsTable.Row>
        <DetailsTable.Row label="Proxy endpoint">
          <code>{`${window.location.protocol}//${proxyPath}/`}</code>
        </DetailsTable.Row>
      </DetailsTable>

      <div className="border-border rounded border border-2 border-solid p-4 text-sm">
        <h5 className="mb-2 font-medium">How to use</h5>
        <p className="mb-1">
          Authenticate with your Portainer username and password:
        </p>
        <pre className="mb-2 whitespace-pre-wrap break-all">
          {`docker login ${window.location.host}`}
        </pre>
        <p className="mb-1">Pull an image through the proxy:</p>
        <pre className="mb-2 whitespace-pre-wrap break-all">
          {`docker pull ${proxyPath}/<image>:<tag>`}
        </pre>
        <p className="mb-1">Push an image (Portainer admins only):</p>
        <pre className="whitespace-pre-wrap break-all">
          {`docker push ${proxyPath}/<image>:<tag>`}
        </pre>
      </div>
    </div>
  );
}
