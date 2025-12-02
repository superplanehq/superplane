import { useCallback, useEffect, useState } from "react";
import { getIntegrationConfig } from "./integrationConfigs";
import type { FormErrors, IntegrationData } from "./types";

interface UseIntegrationFormProps {
  integrationType: string;
  integrations: any[];
  editingIntegration?: any;
}

export const NEW_SECRET_NAME = "my-api-token";

export function useIntegrationForm({ integrationType, integrations, editingIntegration }: UseIntegrationFormProps) {
  const [integrationData, setIntegrationData] = useState<IntegrationData>(() => {
    if (editingIntegration) {
      return {
        name: editingIntegration.metadata?.name || "",
        orgUrl: editingIntegration.spec?.url || "",
        apiToken: {
          secretName: editingIntegration.spec?.auth?.token?.valueFrom?.secret?.name || "",
          secretKey: editingIntegration.spec?.auth?.token?.valueFrom?.secret?.key || "",
        },
      };
    }
    return {
      orgUrl: "",
      name: "",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    };
  });

  const [secretValue, setSecretValue] = useState("");
  const [errors, setErrors] = useState<FormErrors>({});

  // Sync form data when editingIntegration changes
  useEffect(() => {
    if (editingIntegration) {
      setIntegrationData({
        name: editingIntegration.metadata?.name || "",
        orgUrl: editingIntegration.spec?.url || "",
        apiToken: {
          secretName: editingIntegration.spec?.auth?.token?.valueFrom?.secret?.name || "",
          secretKey: editingIntegration.spec?.auth?.token?.valueFrom?.secret?.key || "",
        },
      });
      setSecretValue("");
      setErrors({});
    } else {
      setIntegrationData({
        orgUrl: "",
        name: "",
        apiToken: {
          secretName: "",
          secretKey: "",
        },
      });
      setSecretValue("");
      setErrors({});
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingIntegration?.metadata?.id]);

  const config = getIntegrationConfig(integrationType);

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    const urlError = config.validateUrl(integrationData.orgUrl);
    if (urlError) {
      newErrors.orgUrl = urlError;
    }

    if (!integrationData.name.trim()) {
      newErrors.name = "Field cannot be empty";
    } else {
      // Check for duplicate names, but exclude the currently editing integration
      const isDuplicate = integrations.some(
        (int) =>
          int.metadata?.name === integrationData.name.trim() && int.metadata?.id !== editingIntegration?.metadata?.id,
      );
      if (isDuplicate) {
        newErrors.name = "Integration with this name already exists";
      }
    }

    if (!secretValue.trim()) {
      newErrors.secretValue = "Field cannot be empty";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const resetForm = useCallback(() => {
    setIntegrationData({
      orgUrl: "",
      name: "",
      apiToken: { secretName: "", secretKey: "" },
    });
    setSecretValue("");
    setErrors({});
  }, []);

  return {
    integrationData,
    setIntegrationData,
    secretValue,
    setSecretValue,
    errors,
    setErrors,
    validateForm,
    resetForm,
    config,
  };
}
