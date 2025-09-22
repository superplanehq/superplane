import { Field, Label, ErrorMessage } from '../Fieldset/fieldset';
import { Input } from '../Input/input';
import { Text } from '../Text/text';
import { Link } from '../Link/link';
import type { IntegrationData, FormErrors } from './types';
import { useSecret } from '../../pages/canvas/hooks/useSecrets';

interface SettingsApiTokenFormProps {
  integrationData: IntegrationData;
  setIntegrationData: React.Dispatch<React.SetStateAction<IntegrationData>>;
  errors: FormErrors;
  setErrors: React.Dispatch<React.SetStateAction<FormErrors>>;
  secrets: any[];
  canvasId: string;
  organizationId: string;
  isEditMode?: boolean;
  newSecretValue?: string;
  setNewSecretValue?: React.Dispatch<React.SetStateAction<string>>;
}

export function SettingsApiTokenForm({
  integrationData,
  setIntegrationData,
  errors,
  setErrors,
  secrets,
  canvasId,
  organizationId,
  isEditMode = false,
  newSecretValue = '',
  setNewSecretValue
}: SettingsApiTokenFormProps) {
  const { data: selectedSecret } = useSecret(
    canvasId,
    "DOMAIN_TYPE_CANVAS",
    integrationData.apiToken.secretName
  );


  // In edit mode, show a simplified form that updates the secret value directly
  if (isEditMode && integrationData.apiToken.secretName) {
    return (
      <div className="space-y-4">
        <div className="text-sm font-medium text-gray-900 dark:text-white">
          API Token
        </div>

        <Field>
          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
            Update Token Value
          </Label>
          <Input
            type="password"
            placeholder="Enter new API token value"
            value={newSecretValue}
            className="w-full"
            onChange={(e) => {
              if (setNewSecretValue) {
                setNewSecretValue(e.target.value);
              }
              if (errors.secretValue) {
                setErrors(prev => ({ ...prev, secretValue: undefined }));
              }
            }}
          />
          <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
            Updating this value will modify the secret <strong>{integrationData.apiToken.secretName}</strong> in your canvas secrets.
          </Text>
          {errors.secretValue && <ErrorMessage>{errors.secretValue}</ErrorMessage>}
        </Field>
      </div>
    );
  }

  // For non-edit mode, fall back to the regular secret selection
  return (
    <div className="space-y-4">
      <div className="text-sm font-medium text-gray-900 dark:text-white">
        API Token
      </div>

      <Field>
        <Label className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
          Select Secret
        </Label>
        <select
          value={integrationData.apiToken.secretName}
          onChange={(e) => {
            setIntegrationData(prev => ({
              ...prev,
              apiToken: {
                secretName: e.target.value,
                secretKey: '' // Reset key selection when secret changes
              }
            }));
            if (errors.apiToken) {
              setErrors(prev => ({ ...prev, apiToken: undefined }));
            }
          }}
          className="mt-2 block w-full rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="">Choose a secret...</option>
          {secrets.map((secret) => (
            <option key={secret.metadata?.id} value={secret.metadata?.name}>
              {secret.metadata?.name}
            </option>
          ))}
        </select>
        {secrets.length === 0 && (
          <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
            No secrets available. Create a secret first in the &nbsp;
            <Link
              href={`/${organizationId}/canvas/${canvasId}#secrets`}
              className="text-blue-600 hover:underline"
            >
              secrets section
            </Link>.
          </Text>
        )}
      </Field>

      {/* Key Selection (if secret is selected) */}
      {integrationData.apiToken.secretName && selectedSecret && (
        <Field>
          <Label className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
            Select Key
          </Label>
          <select
            value={integrationData.apiToken.secretKey}
            onChange={(e) => {
              setIntegrationData(prev => ({
                ...prev,
                apiToken: { ...prev.apiToken, secretKey: e.target.value }
              }));
              if (errors.apiToken) {
                setErrors(prev => ({ ...prev, apiToken: undefined }));
              }
            }}
            className="mt-2 block w-full rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="">Choose a key...</option>
            {selectedSecret.spec?.local?.data && Object.keys(selectedSecret.spec.local.data).map((key) => (
              <option key={key} value={key}>
                {key}
              </option>
            ))}
          </select>
          <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
            Select which key from the secret to use as the authentication token.
          </Text>
        </Field>
      )}

      {errors.apiToken && <ErrorMessage>{errors.apiToken}</ErrorMessage>}
    </div>
  );
}