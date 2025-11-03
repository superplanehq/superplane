import { createPortal } from 'react-dom';
import { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogActions,
} from '../Dialog/dialog';
import { Button } from '../Button/button';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { Link } from '../Link/link';
import { useIntegrations, useCreateIntegration } from '@/hooks/useIntegrations';
import { useSecrets, useCreateSecret } from '@/hooks/useSecrets';
import { GitHubIntegrationForm, SemaphoreIntegrationForm, ApiTokenForm, NEW_SECRET_NAME, useIntegrationForm } from '../IntegrationForm';

interface IntegrationModalProps {
  open: boolean;
  onClose: () => void;
  integrationType: string;
  canvasId: string;
  organizationId: string;
  onSuccess?: (integrationId: string) => void;
}


export function IntegrationModal({
  open,
  onClose,
  integrationType,
  canvasId,
  organizationId: organizationId,
  onSuccess
}: IntegrationModalProps) {
  const [isCreating, setIsCreating] = useState(false);
  const orgUrlRef = useRef<HTMLInputElement>(null);

  const { data: integrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");
  const { data: secrets = [] } = useSecrets(canvasId, "DOMAIN_TYPE_CANVAS");
  const createIntegrationMutation = useCreateIntegration(canvasId, "DOMAIN_TYPE_CANVAS");
  const createSecretMutation = useCreateSecret(canvasId, "DOMAIN_TYPE_CANVAS");

  const {
    integrationData,
    setIntegrationData,
    apiTokenTab,
    setApiTokenTab,
    newSecretToken,
    setNewSecretToken,
    errors,
    setErrors,
    validateForm,
    resetForm,
    config
  } = useIntegrationForm({ integrationType, integrations });

  useEffect(() => {
    if (open && orgUrlRef.current) {
      setTimeout(() => {
        orgUrlRef.current?.focus();
      }, 100);
    }
  }, [open]);


  const handleSaveIntegration = async () => {
    if (!validateForm()) {
      return;
    }

    setIsCreating(true);

    try {
      let secretName = integrationData.apiToken.secretName;
      let secretKey = integrationData.apiToken.secretKey;

      if (apiTokenTab === 'new') {
        try {
          const secretData = {
            name: `${integrationData.name.trim()}-api-key`,
            environmentVariables: [{
              name: NEW_SECRET_NAME,
              value: newSecretToken
            }]
          };

          await createSecretMutation.mutateAsync(secretData);
          secretName = secretData.name;
          secretKey = NEW_SECRET_NAME;
        } catch {
          setErrors({ apiToken: 'Failed to create a secret, please try to create secret manually and import' });
          return;
        }
      }

      let trimmedUrl = integrationData.orgUrl.trim();

      if (trimmedUrl.endsWith('/')) {
        trimmedUrl = trimmedUrl.slice(0, -1);
      }

      const integrationPayload = {
        name: integrationData.name.trim(),
        type: integrationType,
        url: trimmedUrl,
        authType: 'AUTH_TYPE_TOKEN' as const,
        tokenSecretName: secretName,
        tokenSecretKey: secretKey
      };

      const result = await createIntegrationMutation.mutateAsync(integrationPayload);

      onSuccess?.(result.data?.integration?.metadata?.id || '');
      onClose();
      resetForm();
    } catch (error) {
      console.error('Failed to create integration:', error);
    } finally {
      setIsCreating(false);
    }
  };


  if (!open) return null;

  return createPortal(
    <Dialog
      open={open}
      onClose={() => { }}
      className="relative z-50"
      size="md"
    >
      <DialogTitle>Create {config.displayName} Integration</DialogTitle>
      <DialogDescription className='text-sm'>
        New integration will be saved to integrations page. Manage integrations{' '}
        <Link href={`/${organizationId}/canvas/${canvasId}#integrations`} className='text-blue-600 dark:text-blue-400'> here</Link>.
      </DialogDescription>

      <DialogBody className="space-y-6">
        {integrationType === 'github' ? (
          <GitHubIntegrationForm
            integrationData={integrationData}
            setIntegrationData={setIntegrationData}
            errors={errors}
            setErrors={setErrors}
            apiTokenTab={apiTokenTab}
            setApiTokenTab={setApiTokenTab}
            newSecretToken={newSecretToken}
            setNewSecretToken={setNewSecretToken}
            secrets={secrets}
            orgUrlRef={orgUrlRef}
          />
        ) : (
          <SemaphoreIntegrationForm
            integrationData={integrationData}
            setIntegrationData={setIntegrationData}
            errors={errors}
            setErrors={setErrors}
            apiTokenTab={apiTokenTab}
            setApiTokenTab={setApiTokenTab}
            newSecretToken={newSecretToken}
            setNewSecretToken={setNewSecretToken}
            secrets={secrets}
            orgUrlRef={orgUrlRef}
          />
        )}

        <ApiTokenForm
          integrationData={integrationData}
          setIntegrationData={setIntegrationData}
          errors={errors}
          setErrors={setErrors}
          apiTokenTab={apiTokenTab}
          setApiTokenTab={setApiTokenTab}
          newSecretToken={newSecretToken}
          setNewSecretToken={setNewSecretToken}
          secrets={secrets}
          orgUrlRef={orgUrlRef}
          organizationId={organizationId}
          canvasId={canvasId}
        />
      </DialogBody>

      <DialogActions>
        <Button
          onClick={onClose}
          disabled={isCreating}
        >
          Cancel
        </Button>
        <Button
          color='blue'
          onClick={handleSaveIntegration}
          disabled={isCreating}
        >
          {isCreating ? (
            <>
              <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
              Creating...
            </>
          ) : (
            'Create'
          )}
        </Button>
      </DialogActions>
    </Dialog>,
    document.body
  );
}