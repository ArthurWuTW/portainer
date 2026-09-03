import {
  Form,
  Formik,
  FormikHelpers,
  FormikProps,
  useFormikContext,
} from 'formik';
import { object, string } from 'yup';
import { useRef } from 'react';

import { useEnvironmentList } from '@/react/portainer/environments/queries';
import { useGroups } from '@/react/portainer/environments/environment-groups/queries';
import {
  EnvironmentId,
  EnvironmentGroupId,
} from '@/react/portainer/environments/types';
import { useCanExit } from '@/react/hooks/useCanExit';

import { FormControl } from '@@/form-components/FormControl';
import { Input } from '@@/form-components/Input';
import { PortainerSelect } from '@@/form-components/PortainerSelect';
import { CodeEditor } from '@@/CodeEditor';
import { LoadingButton } from '@@/buttons';
import { confirmGenericDiscard } from '@@/modals/confirm';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

export interface ServiceInstanceFormValues {
  name: string;
  description: string;
  targetMode: 'group' | 'environments';
  groupId: EnvironmentGroupId | null;
  environmentIds: EnvironmentId[];
  composeFile: string;
}

interface Props {
  initialValues: ServiceInstanceFormValues;
  onSubmit: (
    values: ServiceInstanceFormValues,
    helpers: FormikHelpers<ServiceInstanceFormValues>
  ) => Promise<void>;
  submitLabel: string;
  submitLoadingLabel: string;
}

const validationSchema = object({
  name: string().required('Name is required'),
  description: string(),
  targetMode: string().required(),
  composeFile: string().required('Compose file is required'),
});

export function ServiceInstanceForm({
  initialValues,
  onSubmit,
  submitLabel,
  submitLoadingLabel,
}: Props) {
  const formikRef = useRef<FormikProps<ServiceInstanceFormValues>>(null);
  useCanExit(() => !formikRef.current?.dirty || confirmGenericDiscard());

  return (
    <Formik
      innerRef={formikRef}
      initialValues={initialValues}
      onSubmit={onSubmit}
      validationSchema={validationSchema}
      validateOnMount
      enableReinitialize
    >
      <InnerForm
        submitLabel={submitLabel}
        submitLoadingLabel={submitLoadingLabel}
      />
    </Formik>
  );
}

interface InnerFormProps {
  submitLabel: string;
  submitLoadingLabel: string;
}

function InnerForm({ submitLabel, submitLoadingLabel }: InnerFormProps) {
  const {
    values,
    errors,
    handleChange,
    setFieldValue,
    isValid,
    dirty,
    isSubmitting,
  } = useFormikContext<ServiceInstanceFormValues>();

  const groupsQuery = useGroups();
  const environmentsQuery = useEnvironmentList(
    { pageLimit: 0, excludeSnapshots: true },
    { enabled: values.targetMode === 'environments' }
  );

  const groupOptions = (groupsQuery.data ?? []).map((g) => ({
    value: g.Id,
    label: g.Name,
  }));

  const environmentOptions = (environmentsQuery.environments ?? []).map(
    (e) => ({
      value: e.Id,
      label: e.Name,
    })
  );

  return (
    <Form className="form-horizontal">
      <FormControl label="Name" required errors={errors.name} inputId="si-name">
        <Input
          id="si-name"
          name="name"
          value={values.name}
          onChange={handleChange}
          placeholder="e.g. production-web"
          data-cy="service-instance-name-input"
        />
      </FormControl>

      <FormControl
        label="Description"
        errors={errors.description}
        inputId="si-description"
      >
        <Input
          id="si-description"
          name="description"
          value={values.description}
          onChange={handleChange}
          placeholder="e.g. production web service"
          data-cy="service-instance-description-input"
        />
      </FormControl>

      <FormControl label="Target mode" required>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2">
            <input
              type="radio"
              name="targetMode"
              value="group"
              checked={values.targetMode === 'group'}
              onChange={() => setFieldValue('targetMode', 'group')}
              data-cy="service-instance-target-group-radio"
            />
            <span>Environment Group</span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="radio"
              name="targetMode"
              value="environments"
              checked={values.targetMode === 'environments'}
              onChange={() => setFieldValue('targetMode', 'environments')}
              data-cy="service-instance-target-envs-radio"
            />
            <span>Individual Environments</span>
          </label>
        </div>
      </FormControl>

      {values.targetMode === 'group' && (
        <FormControl label="Group" required inputId="si-group">
          <PortainerSelect
            inputId="si-group"
            name="groupId"
            value={values.groupId ?? undefined}
            onChange={(val) => setFieldValue('groupId', val ?? null)}
            options={groupOptions}
            placeholder="Select a group"
            isLoading={groupsQuery.isLoading}
            data-cy="service-instance-group-select"
          />
        </FormControl>
      )}

      {values.targetMode === 'environments' && (
        <FormControl label="Environments" required inputId="si-environments">
          <PortainerSelect<EnvironmentId>
            inputId="si-environments"
            name="environmentIds"
            isMulti
            value={values.environmentIds}
            onChange={(val) => setFieldValue('environmentIds', val)}
            options={environmentOptions}
            placeholder="Select environments"
            isLoading={environmentsQuery.isLoading}
            data-cy="service-instance-env-select"
          />
        </FormControl>
      )}

      <FormControl label="Compose file" required errors={errors.composeFile}>
        <CodeEditor
          id="si-compose"
          value={values.composeFile}
          onChange={(val) => setFieldValue('composeFile', val)}
          height="300px"
          data-cy="service-instance-compose-editor"
        />
      </FormControl>

      <StickyFooter className="justify-end gap-4">
        <LoadingButton
          size="medium"
          loadingText={submitLoadingLabel}
          isLoading={isSubmitting}
          disabled={!isValid || isSubmitting || !dirty}
          data-cy="service-instance-submit-button"
        >
          {submitLabel}
        </LoadingButton>
      </StickyFooter>
    </Form>
  );
}
