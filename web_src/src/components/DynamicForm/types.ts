import type { SuperplaneFieldManifest } from '../../api-client';

export interface DynamicFormContext {
  integrationName?: string;
  organizationId?: string;
  canvasId?: string;
}

export interface DynamicFormFieldProps {
  field: SuperplaneFieldManifest;
  value: any;
  onChange: (value: any) => void;
  parentPath?: string;
  disabled?: boolean;
  error?: string;
  context?: DynamicFormContext;
}

export interface MapFieldProps {
  value: Record<string, string>;
  onChange: (value: Record<string, string>) => void;
  placeholder?: string;
  disabled?: boolean;
}

export interface ArrayFieldProps {
  field: SuperplaneFieldManifest;
  value: any[];
  onChange: (value: any[]) => void;
  disabled?: boolean;
  parentPath?: string;
  context?: DynamicFormContext;
}
