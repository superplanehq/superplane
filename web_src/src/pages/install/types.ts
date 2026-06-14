export interface OrganizationOption {
  id: string;
  name: string;
}

export interface InstallParam {
  name: string;
  label: string;
  type: string;
  placeholder?: string;
  description?: string;
  default?: string;
  required: boolean;
}

export interface InstallPreview {
  repo: string;
  title: string;
  description?: string;
  canvasName?: string;
  defaultName: string;
  installParams?: InstallParam[];
  integrations?: string[];
}

export interface InstallResult {
  canvasId: string;
  organizationId: string;
}
