export interface OrganizationOption {
  id: string;
  name: string;
}

export interface InstallPreview {
  repo: string;
  title: string;
  description?: string;
  defaultName: string;
}

export interface InstallResult {
  canvasId: string;
  organizationId: string;
}
