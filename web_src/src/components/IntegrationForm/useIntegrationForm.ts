import { useState } from 'react';
import type { FormErrors, IntegrationData } from './types';
import { getIntegrationConfig } from './integrationConfigs';

interface UseIntegrationFormProps {
  integrationType: string;
  integrations: any[];
  editingIntegration?: any;
}

export const NEW_SECRET_NAME = 'my-api-token';

export function useIntegrationForm({ integrationType, integrations, editingIntegration }: UseIntegrationFormProps) {
  const [integrationData, setIntegrationData] = useState<IntegrationData>({
    orgUrl: '',
    name: '',
    apiToken: {
      secretName: '',
      secretKey: ''
    }
  });
  
  const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new');
  const [newSecretToken, setNewSecretToken] = useState('');
  const [errors, setErrors] = useState<FormErrors>({});

  const config = getIntegrationConfig(integrationType);

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};
    
    const urlError = config.validateUrl(integrationData.orgUrl);
    if (urlError) {
      newErrors.orgUrl = urlError;
    }
    
    if (!integrationData.name.trim()) {
      newErrors.name = 'Field cannot be empty';
    } else {
      // Check for duplicate names, but exclude the currently editing integration
      const isDuplicate = integrations.some(int =>
        int.metadata?.name === integrationData.name.trim() &&
        int.metadata?.id !== editingIntegration?.metadata?.id
      );
      if (isDuplicate) {
        newErrors.name = 'Integration with this name already exists';
      }
    }
    
    if (apiTokenTab === 'new') {
      if (!NEW_SECRET_NAME.trim()) {
        newErrors.secretName = 'Field cannot be empty';
      }
      if (!newSecretToken.trim()) {
        newErrors.secretValue = 'Field cannot be empty';
      }
    } else {
      if (!integrationData.apiToken.secretName || !integrationData.apiToken.secretKey) {
        newErrors.apiToken = 'Please select a secret and key';
      }
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const resetForm = () => {
    setIntegrationData({
      orgUrl: '',
      name: '',
      apiToken: { secretName: '', secretKey: '' }
    });
    setNewSecretToken('');
    setApiTokenTab('new');
    setErrors({});
  };

  return {
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
  };
}