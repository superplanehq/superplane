import { createPortal } from "react-dom";
import { useState, useEffect, useRef } from "react";
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from "../Dialog/dialog";
import { Button } from "../Button/button";
import { MaterialSymbol } from "../MaterialSymbol/material-symbol";
import { Link } from "../Link/link";
import { useIntegrations, useCreateIntegration, useUpdateIntegration } from "@/hooks/useIntegrations";
import { useSecrets, useCreateSecret } from "@/hooks/useSecrets";
import {
  GitHubIntegrationForm,
  SemaphoreIntegrationForm,
  ApiTokenForm,
  NEW_SECRET_NAME,
  useIntegrationForm,
} from "../IntegrationForm";
import type { IntegrationsIntegration } from "@/api-client/types.gen";

interface IntegrationModalProps {
  open: boolean;
  onClose: () => void;
  integrationType: string;
  canvasId: string;
  organizationId: string;
  onSuccess?: (integrationId: string) => void;
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION";
  editingIntegration?: IntegrationsIntegration;
}

export function IntegrationModal({
  open,
  onClose,
  integrationType,
  canvasId,
  organizationId: organizationId,
  onSuccess,
  domainType = "DOMAIN_TYPE_CANVAS",
  editingIntegration,
}: IntegrationModalProps) {
  const [isCreating, setIsCreating] = useState(false);
  const orgUrlRef = useRef<HTMLInputElement>(null);

  const { data: integrations = [] } = useIntegrations(organizationId, domainType);
  const { data: secrets = [] } = useSecrets(organizationId, domainType);
  const createIntegrationMutation = useCreateIntegration(organizationId, domainType);
  const updateIntegrationMutation = useUpdateIntegration(
    organizationId,
    domainType,
    editingIntegration?.metadata?.id || "",
  );
  const createSecretMutation = useCreateSecret(organizationId, domainType);

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
    config,
  } = useIntegrationForm({ integrationType, integrations, editingIntegration });

  useEffect(() => {
    if (open && orgUrlRef.current) {
      setTimeout(() => {
        orgUrlRef.current?.focus();
      }, 100);
    }
  }, [open]);

  // Clear form when modal is closed
  useEffect(() => {
    if (!open) {
      resetForm();
      setErrors({});
    }
  }, [open, resetForm, setErrors]);

  // Reset form when modal is opened for creating new integration
  useEffect(() => {
    if (open && !editingIntegration) {
      resetForm();
    }
  }, [open, editingIntegration, resetForm]);

  const handleSaveIntegration = async () => {
    if (!validateForm()) {
      return;
    }

    setIsCreating(true);
    const isEditing = !!editingIntegration;

    try {
      let secretName = integrationData.apiToken.secretName;
      let secretKey = integrationData.apiToken.secretKey;

      if (apiTokenTab === "new") {
        try {
          let newSecretName = `${integrationData.name.trim()}-api-key`;
          const conflictingSecretsCount = secrets.reduce((acc, secret) => {
            if (secret.metadata?.name === newSecretName) {
              return acc + 1;
            }
            return acc;
          }, 0);

          if (conflictingSecretsCount > 0) {
            newSecretName = `${newSecretName}-${conflictingSecretsCount + 1}`;
          }

          const secretData = {
            name: newSecretName,
            environmentVariables: [
              {
                name: NEW_SECRET_NAME,
                value: newSecretToken,
              },
            ],
          };

          await createSecretMutation.mutateAsync(secretData);
          secretName = secretData.name;
          secretKey = NEW_SECRET_NAME;
        } catch {
          setErrors({ apiToken: "Failed to create a secret, please try to create secret manually and import" });
          return;
        }
      }

      let trimmedUrl = integrationData.orgUrl.trim();

      if (trimmedUrl.endsWith("/")) {
        trimmedUrl = trimmedUrl.slice(0, -1);
      }

      const integrationPayload = {
        name: integrationData.name.trim(),
        type: integrationType,
        url: trimmedUrl,
        authType: "AUTH_TYPE_TOKEN" as const,
        tokenSecretName: secretName,
        tokenSecretKey: secretKey,
      };

      let result;
      if (isEditing) {
        result = await updateIntegrationMutation.mutateAsync({
          id: editingIntegration?.metadata?.id || "",
          ...integrationPayload,
        });
      } else {
        result = await createIntegrationMutation.mutateAsync(integrationPayload);
      }

      onSuccess?.(result.data?.integration?.metadata?.id || "");
      onClose();
      resetForm();
    } catch (error) {
      console.error(`Failed to ${isEditing ? "update" : "create"} integration:`, error);
    } finally {
      setIsCreating(false);
    }
  };

  if (!open) return null;

  return createPortal(
    <Dialog open={open} onClose={() => {}} className="relative z-50" size="md">
      <DialogTitle>
        {editingIntegration ? "Edit" : "Create"} {config.displayName} Integration
      </DialogTitle>
      <DialogDescription className="text-sm">
        {editingIntegration
          ? "Update your integration settings below."
          : "New integration will be saved to integrations page."}{" "}
        Manage integrations{" "}
        <Link
          href={
            domainType === "DOMAIN_TYPE_ORGANIZATION"
              ? `/${organizationId}/settings/integrations`
              : `/${organizationId}/canvas/${canvasId}#integrations`
          }
          className="text-blue-600 dark:text-blue-400"
        >
          {" "}
          here
        </Link>
        .
      </DialogDescription>

      <DialogBody className="space-y-6">
        {integrationType === "github" ? (
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
        <Button onClick={onClose} disabled={isCreating}>
          Cancel
        </Button>
        <Button color="blue" onClick={handleSaveIntegration} disabled={isCreating}>
          {isCreating ? (
            <>
              <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
              {editingIntegration ? "Updating..." : "Creating..."}
            </>
          ) : editingIntegration ? (
            "Update"
          ) : (
            "Create"
          )}
        </Button>
      </DialogActions>
    </Dialog>,
    document.body,
  );
}
