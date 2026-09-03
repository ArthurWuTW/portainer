import { CodeEditor } from '@@/CodeEditor';
import { Widget } from '@@/Widget';

import { ServiceInstance } from '../types';

interface Props {
  instance: ServiceInstance;
}

export function ComposeTab({ instance }: Props) {
  return (
    <Widget>
      <Widget.Title title="Compose" />
      <Widget.Body>
        <CodeEditor
          id="service-instance-compose"
          value={instance.ComposeFile}
          readonly
          height="400px"
          data-cy="service-instance-compose-editor"
        />
      </Widget.Body>
    </Widget>
  );
}
