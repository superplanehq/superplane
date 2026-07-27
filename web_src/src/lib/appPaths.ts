const RESERVED_APP_PATH_SEGMENTS = new Set(["new"]);

export function isAppRouteId(segment: string | undefined | null): segment is string {
  return !!segment && !RESERVED_APP_PATH_SEGMENTS.has(segment);
}

export function appPath(organizationId: string, appId: string, search = ""): string {
  return `/${organizationId}/apps/${appId}${search}`;
}

export function appRunPath(organizationId: string, appId: string, runId: string): string {
  return appPath(organizationId, appId, `?run=${runId}`);
}

const APP_CANVAS_PATH = /^\/[^/]+\/apps\/[^/?#]+$/;

export function parseAppRunPath(value: string): string | null {
  try {
    const url = value.startsWith("/") ? new URL(value, "http://local.test") : new URL(value);
    if (!APP_CANVAS_PATH.test(url.pathname) || !url.searchParams.get("run")) {
      return null;
    }

    if (!value.startsWith("/") && typeof window !== "undefined" && url.origin !== window.location.origin) {
      return null;
    }

    return `${url.pathname}${url.search}`;
  } catch {
    return null;
  }
}

export function appSettingsPath(organizationId: string, appId: string): string {
  return `/${organizationId}/apps/${appId}/settings`;
}
