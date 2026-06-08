const RESERVED_APP_PATH_SEGMENTS = new Set(["new"]);

export function isAppRouteId(segment: string | undefined | null): segment is string {
  return !!segment && !RESERVED_APP_PATH_SEGMENTS.has(segment);
}

export function appPath(organizationId: string, appId: string, search = ""): string {
  return `/${organizationId}/apps/${appId}${search}`;
}

export function appSettingsPath(organizationId: string, appId: string): string {
  return `/${organizationId}/apps/${appId}/settings`;
}
