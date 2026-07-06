export const CANVAS_APP_PREFERENCES_STORAGE_KEY = "canvas-app-preferences-v1";

export type CanvasAppPreferences = {
  pinnedCanvasIds: string[];
  starredCanvasIds: string[];
};

type PreferencesByScope = Record<string, CanvasAppPreferences>;

const EMPTY_PREFERENCES: CanvasAppPreferences = {
  pinnedCanvasIds: [],
  starredCanvasIds: [],
};

function preferenceScope(organizationId: string, accountId?: string): string {
  return `${organizationId}:${accountId || "anonymous"}`;
}

function sanitizeCanvasIds(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter((id): id is string => typeof id === "string" && id.trim().length > 0);
}

function sanitizePreferences(value: unknown): CanvasAppPreferences {
  if (!value || typeof value !== "object") {
    return EMPTY_PREFERENCES;
  }

  const record = value as Record<string, unknown>;
  return {
    pinnedCanvasIds: sanitizeCanvasIds(record.pinnedCanvasIds),
    starredCanvasIds: sanitizeCanvasIds(record.starredCanvasIds),
  };
}

function readAllPreferences(): PreferencesByScope {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(CANVAS_APP_PREFERENCES_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object") {
      return {};
    }

    const result: PreferencesByScope = {};
    for (const [scope, preferences] of Object.entries(parsed as Record<string, unknown>)) {
      result[scope] = sanitizePreferences(preferences);
    }

    return result;
  } catch {
    return {};
  }
}

function writeAllPreferences(preferences: PreferencesByScope): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(CANVAS_APP_PREFERENCES_STORAGE_KEY, JSON.stringify(preferences));
  } catch {
    // App-list preferences are optional.
  }
}

export function loadCanvasAppPreferences(organizationId: string, accountId?: string): CanvasAppPreferences {
  if (!organizationId) {
    return EMPTY_PREFERENCES;
  }

  return readAllPreferences()[preferenceScope(organizationId, accountId)] ?? EMPTY_PREFERENCES;
}

export function setCanvasPinned(
  organizationId: string,
  accountId: string | undefined,
  canvasId: string,
  pinned: boolean,
) {
  return updateCanvasPreferences(organizationId, accountId, (preferences) => ({
    ...preferences,
    pinnedCanvasIds: updateOrderedCanvasIds(preferences.pinnedCanvasIds, canvasId, pinned),
  }));
}

export function setCanvasStarred(
  organizationId: string,
  accountId: string | undefined,
  canvasId: string,
  starred: boolean,
) {
  return updateCanvasPreferences(organizationId, accountId, (preferences) => ({
    ...preferences,
    starredCanvasIds: updateOrderedCanvasIds(preferences.starredCanvasIds, canvasId, starred),
  }));
}

function updateCanvasPreferences(
  organizationId: string,
  accountId: string | undefined,
  update: (preferences: CanvasAppPreferences) => CanvasAppPreferences,
): CanvasAppPreferences {
  if (!organizationId) {
    return EMPTY_PREFERENCES;
  }

  const allPreferences = readAllPreferences();
  const scope = preferenceScope(organizationId, accountId);
  const next = update(allPreferences[scope] ?? EMPTY_PREFERENCES);
  allPreferences[scope] = next;
  writeAllPreferences(allPreferences);
  return next;
}

function updateOrderedCanvasIds(canvasIds: string[], canvasId: string, enabled: boolean): string[] {
  if (!canvasId) {
    return canvasIds;
  }

  const withoutCanvas = canvasIds.filter((id) => id !== canvasId);
  return enabled ? [canvasId, ...withoutCanvas] : withoutCanvas;
}
