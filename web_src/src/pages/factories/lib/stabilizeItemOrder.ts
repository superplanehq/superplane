/**
 * Keeps the previous on-screen order when the same items come back with
 * new fields. One new item goes first. Many new items keep the incoming
 * sort, so a filter change can rebuild the list.
 */
export function stabilizeItemOrder<T>(
  previousIds: readonly string[],
  items: T[],
  idOf: (item: T) => string | undefined,
): T[] {
  if (previousIds.length === 0) {
    return items;
  }

  const byId = indexById(items, idOf);
  if (addedIdCount(previousIds, byId) > 1) {
    return items;
  }
  return mergePreviousOrder(previousIds, items, byId, idOf);
}

function indexById<T>(items: T[], idOf: (item: T) => string | undefined): Map<string, T> {
  const byId = new Map<string, T>();
  for (const item of items) {
    const id = idOf(item);
    if (id) {
      byId.set(id, item);
    }
  }
  return byId;
}

function addedIdCount<T>(previousIds: readonly string[], byId: Map<string, T>): number {
  const previous = new Set(previousIds);
  let added = 0;
  for (const id of byId.keys()) {
    if (!previous.has(id)) {
      added += 1;
    }
  }
  return added;
}

function mergePreviousOrder<T>(
  previousIds: readonly string[],
  items: T[],
  byId: Map<string, T>,
  idOf: (item: T) => string | undefined,
): T[] {
  const seen = new Set<string>();
  const kept: T[] = [];
  for (const id of previousIds) {
    const item = byId.get(id);
    if (!item) {
      continue;
    }
    kept.push(item);
    seen.add(id);
  }

  const newcomers = items.filter((item) => {
    const id = idOf(item);
    if (!id) {
      return false;
    }
    return !seen.has(id);
  });
  return [...newcomers, ...kept];
}
