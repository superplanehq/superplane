export interface FormErrors {
  orgUrl?: string;
  name?: string;
  apiToken?: string;
  secretName?: string;
  secretValue?: string;
}

export interface IntegrationData {
  orgUrl: string;
  name: string;
  apiToken: {
    secretName: string;
    secretKey: string;
  };
}

export interface BaseIntegrationFormProps {
  integrationData: IntegrationData;
  setIntegrationData: React.Dispatch<React.SetStateAction<IntegrationData>>;
  errors: FormErrors;
  setErrors: React.Dispatch<React.SetStateAction<FormErrors>>;
  secretValue: string;
  setSecretValue: React.Dispatch<React.SetStateAction<string>>;
  orgUrlRef: React.RefObject<HTMLInputElement | null>;
}

export interface IntegrationConfig {
  displayName: string;
  urlPlaceholder: string;
  orgUrlLabel: string;
  validateUrl: (url: string) => string | undefined;
  extractOrgName: (url: string) => string;
}
