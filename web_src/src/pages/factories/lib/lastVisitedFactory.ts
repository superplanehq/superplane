export const LAST_VISITED_FACTORY_STORAGE_KEY = "superplane:last-visited-factory";

type LastVisitedFactoryByOrg = Record<string, string>;

type LastVisitedFactoryByAccount = Record<string, LastVisitedFactoryByOrg>;

function readAll(): LastVisitedFactoryByAccount {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(LAST_VISITED_FACTORY_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }

    const result: LastVisitedFactoryByAccount = {};
    for (const [accountId, byOrg] of Object.entries(parsed as Record<string, unknown>)) {
      if (!byOrg || typeof byOrg !== "object" || Array.isArray(byOrg)) {
        continue;
      }
      const orgs: LastVisitedFactoryByOrg = {};
      for (const [orgId, factoryId] of Object.entries(byOrg as Record<string, unknown>)) {
        if (typeof factoryId === "string" && factoryId) {
          orgs[orgId] = factoryId;
        }
      }
      if (Object.keys(orgs).length > 0) {
        result[accountId] = orgs;
      }
    }

    return result;
  } catch {
    return {};
  }
}

export function readLastVisitedFactory(accountId: string, organizationId: string): string | null {
  if (!accountId || !organizationId) {
    return null;
  }
  return readAll()[accountId]?.[organizationId] ?? null;
}

/**
 * Picks which workspace to land on from the factories list, preferring the
 * last-visited one (matched by the real `id`, which is what gets persisted —
 * see `recordLastVisitedFactory`). Returns the whole factory (not just the
 * id) so the caller can redirect to its `key`, the canonical routing
 * identifier.
 */
export function pickInitialFactory<T extends { id?: string }>(
  factories: T[],
  lastVisitedFactoryId: string | null,
): T | null {
  const known = factories.filter((factory) => Boolean(factory.id));
  if (known.length === 0) {
    return null;
  }

  if (lastVisitedFactoryId) {
    const lastVisited = known.find((factory) => factory.id === lastVisitedFactoryId);
    if (lastVisited) {
      return lastVisited;
    }
  }

  return known[0];
}

export function recordLastVisitedFactory(accountId: string, organizationId: string, factoryId: string): void {
  if (!accountId || !organizationId || !factoryId || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAll();
    const forAccount = all[accountId] ?? {};
    forAccount[organizationId] = factoryId;
    all[accountId] = forAccount;
    window.localStorage.setItem(LAST_VISITED_FACTORY_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}

export function clearLastVisitedFactory(accountId: string, organizationId: string, factoryId: string): void {
  if (!accountId || !organizationId || !factoryId || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAll();
    const forAccount = all[accountId];
    if (!forAccount || forAccount[organizationId] !== factoryId) {
      return;
    }
    delete forAccount[organizationId];
    if (Object.keys(forAccount).length === 0) {
      delete all[accountId];
    } else {
      all[accountId] = forAccount;
    }
    window.localStorage.setItem(LAST_VISITED_FACTORY_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}
