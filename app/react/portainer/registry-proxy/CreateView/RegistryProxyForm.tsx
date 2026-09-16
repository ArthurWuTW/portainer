import {
  Form,
  Formik,
  FormikHelpers,
  FormikProps,
  useFormikContext,
} from 'formik';
import { object, string } from 'yup';
import { useRef } from 'react';

import { useCanExit } from '@/react/hooks/useCanExit';

import { FormControl } from '@@/form-components/FormControl';
import { Input } from '@@/form-components/Input';
import { SwitchField } from '@@/form-components/SwitchField';
import { LoadingButton } from '@@/buttons';
import { confirmGenericDiscard } from '@@/modals/confirm';
import { StickyFooter } from '@@/StickyFooter/StickyFooter';

export interface RegistryProxyFormValues {
  name: string;
  url: string;
  tls: boolean;
  tlsSkipVerify: boolean;
  authentication: boolean;
  username: string;
  password: string;
}

interface Props {
  initialValues: RegistryProxyFormValues;
  onSubmit: (
    values: RegistryProxyFormValues,
    helpers: FormikHelpers<RegistryProxyFormValues>
  ) => Promise<void>;
  submitLabel: string;
  submitLoadingLabel: string;
  isEditing?: boolean;
}

function getValidationSchema(isEditing: boolean) {
  return object({
    name: string().required('Name is required'),
    url: string().required('Registry URL is required'),
    username: string().when('authentication', {
      is: true,
      then: string().required('Username is required'),
    }),
    password: isEditing
      ? string()
      : string().when('authentication', {
          is: true,
          then: string().required('Password is required'),
        }),
  });
}

export function RegistryProxyForm({
  initialValues,
  onSubmit,
  submitLabel,
  submitLoadingLabel,
  isEditing = false,
}: Props) {
  const formikRef = useRef<FormikProps<RegistryProxyFormValues>>(null);
  useCanExit(() => !formikRef.current?.dirty || confirmGenericDiscard());

  return (
    <Formik
      innerRef={formikRef}
      initialValues={initialValues}
      onSubmit={onSubmit}
      validationSchema={getValidationSchema(isEditing)}
      validateOnMount
      enableReinitialize
    >
      <InnerForm
        submitLabel={submitLabel}
        submitLoadingLabel={submitLoadingLabel}
        isEditing={isEditing}
      />
    </Formik>
  );
}

interface InnerFormProps {
  submitLabel: string;
  submitLoadingLabel: string;
  isEditing: boolean;
}

function InnerForm({
  submitLabel,
  submitLoadingLabel,
  isEditing,
}: InnerFormProps) {
  const { values, errors, handleChange, setFieldValue, isValid, isSubmitting } =
    useFormikContext<RegistryProxyFormValues>();

  return (
    <Form className="form-horizontal">
      <FormControl
        label="Name"
        required
        errors={errors.name}
        inputId="registry-proxy-name"
      >
        <Input
          id="registry-proxy-name"
          name="name"
          value={values.name}
          onChange={handleChange}
          placeholder="e.g. my-local-registry"
          data-cy="registry-proxy-name-input"
        />
      </FormControl>

      <FormControl
        label="Registry URL"
        required
        errors={errors.url}
        inputId="registry-proxy-url"
        tooltip="Host and optional port of the local Docker registry, without scheme. External Docker Hub is not supported."
      >
        <Input
          id="registry-proxy-url"
          name="url"
          value={values.url}
          onChange={handleChange}
          placeholder="e.g. registry.local:5000"
          data-cy="registry-proxy-url-input"
        />
      </FormControl>

      <SwitchField
        label="Use TLS"
        name="tls"
        checked={values.tls}
        onChange={(value) => setFieldValue('tls', value)}
        data-cy="registry-proxy-tls-switch"
      />

      {values.tls && (
        <SwitchField
          label="Skip TLS verification"
          name="tlsSkipVerify"
          checked={values.tlsSkipVerify}
          onChange={(value) => setFieldValue('tlsSkipVerify', value)}
          data-cy="registry-proxy-tls-skip-verify-switch"
        />
      )}

      <SwitchField
        label="Authentication"
        name="authentication"
        checked={values.authentication}
        onChange={(value) => setFieldValue('authentication', value)}
        tooltip="Credentials used by Portainer to contact the local registry. External clients still authenticate with their Portainer account."
        data-cy="registry-proxy-authentication-switch"
      />

      {values.authentication && (
        <>
          <FormControl
            label="Username"
            required
            errors={errors.username}
            inputId="registry-proxy-username"
          >
            <Input
              id="registry-proxy-username"
              name="username"
              value={values.username}
              onChange={handleChange}
              data-cy="registry-proxy-username-input"
            />
          </FormControl>

          <FormControl
            label="Password"
            required={!isEditing}
            errors={errors.password}
            inputId="registry-proxy-password"
            tooltip={
              isEditing
                ? 'Leave blank to keep the current password.'
                : undefined
            }
          >
            <Input
              id="registry-proxy-password"
              name="password"
              type="password"
              value={values.password}
              onChange={handleChange}
              placeholder={isEditing ? 'Unchanged' : undefined}
              data-cy="registry-proxy-password-input"
            />
          </FormControl>
        </>
      )}

      <StickyFooter className="justify-end gap-4">
        <LoadingButton
          size="medium"
          loadingText={submitLoadingLabel}
          isLoading={isSubmitting}
          disabled={!isValid || isSubmitting}
          data-cy="registry-proxy-submit-button"
        >
          {submitLabel}
        </LoadingButton>
      </StickyFooter>
    </Form>
  );
}
