export interface OrganizationOption {
  id: string;
  name: string;
}

export interface InstallParam {
  name: string;
  label: string;
  type: string; // "string" or "integration-resource"
  placeholder?: string;
  description?: string;
  default?: string;
  required: boolean;
  // For type "integration-resource"
  integration?: string; // integration type name (e.g. "digitalocean")
  resourceType?: string; // resource type (e.g. "region", "size", "image")
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
