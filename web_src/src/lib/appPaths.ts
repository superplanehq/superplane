export function appPath(organizationId: string, appId: string, search = ""): string {
  return `/${organizationId}/apps/${appId}${search}`;
}

export function appSettingsPath(organizationId: string, appId: string): string {
  return `/${organizationId}/apps/${appId}/settings`;
}
