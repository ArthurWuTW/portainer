import { ArrowLeft } from 'lucide-react';

import { localizeDate } from '@/react/common/date-utils';
import { humanize } from '@/portainer/filters/filters';
import { trimSHA } from '@/docker/filters/utils';

import { DetailsTable } from '@@/DetailsTable';
import { Alert } from '@@/Alert';
import { Button } from '@@/buttons';
import { DeleteButton } from '@@/buttons/DeleteButton';

import { RegistryProxyId } from '../types';
import { useRegistryProxyTag } from '../queries/useRegistryProxyTag';
import { useDeleteRegistryImage } from '../queries/useRegistryProxyMutations';

interface Props {
  id: RegistryProxyId;
  repository: string;
  tag: string;
  proxyPath: string;
  onBack: () => void;
}

export function TagView({ id, repository, tag, proxyPath, onBack }: Props) {
  const tagQuery = useRegistryProxyTag(id, repository, tag);
  const deleteImageMutation = useDeleteRegistryImage(id, repository);
  const tagData = tagQuery.data;

  if (tagQuery.isLoading) {
    return <span>Loading image version...</span>;
  }

  if (tagQuery.error) {
    return (
      <Alert color="error" title="Unable to load image version">
        {(tagQuery.error as Error).message}
      </Alert>
    );
  }

  if (!tagData) {
    return (
      <Alert color="warn" title="Image version not found">
        {`${repository}:${tag} is not available in this registry.`}
      </Alert>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <Button
          color="link"
          icon={ArrowLeft}
          className="!m-0"
          data-cy="registry-proxy-tag-back-button"
          onClick={onBack}
        >
          Back to {repository}
        </Button>
        <DeleteButton
          size="small"
          text="Delete"
          loadingText="Deleting..."
          confirmMessage={`This will remove '${repository}:${tag}' from the registry. Continue?`}
          onConfirmed={() => {
            deleteImageMutation.mutate(tag);
            onBack();
          }}
          data-cy="registry-proxy-tag-delete-button"
        />
      </div>

      <DetailsTable dataCy="registry-proxy-tag-details">
        <DetailsTable.Row label="Image">{repository}</DetailsTable.Row>
        <DetailsTable.Row label="Version (tag)">
          {tagData.name}
        </DetailsTable.Row>
        <DetailsTable.Row label="Manifest digest">
          <code title={tagData.digest}>{tagData.digest}</code>
        </DetailsTable.Row>
        <DetailsTable.Row label="Manifest size">
          {humanize(tagData.size)}
        </DetailsTable.Row>
        <DetailsTable.Row label="Created">
          {tagData.created ? localizeDate(new Date(tagData.created)) : '-'}
        </DetailsTable.Row>
        <DetailsTable.Row label="Short digest">
          {trimSHA(tagData.digest)}
        </DetailsTable.Row>
        <DetailsTable.Row label="Pull command">
          <code>{`docker pull ${proxyPath}/${repository}:${tagData.name}`}</code>
        </DetailsTable.Row>
      </DetailsTable>
    </div>
  );
}
