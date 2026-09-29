/**
 * Minimal shape shared by connected integrations, used so this helper does not
 * depend on the generated API client types.
 */
export interface ConnectedIntegrationLike {
  metadata?: {
    id?: string;
    name?: string;
    integrationName?: string;
  };
}

type SortKey = readonly [type: string, name: string, id: string];

function toSortKey(integration: ConnectedIntegrationLike): SortKey {
  const type = integration.metadata?.integrationName?.toLowerCase() ?? "";
  const name = integration.metadata?.name || integration.metadata?.integrationName || integration.metadata?.id || "";
  const id = integration.metadata?.id ?? "";
  return [type, name, id];
}

function compareSortKeys(a: SortKey, b: SortKey): number {
  for (let index = 0; index < a.length; index++) {
    const comparison = a[index].localeCompare(b[index]);
    if (comparison !== 0) {
      return comparison;
    }
  }
  return 0;
}

/**
 * Sorts connected integrations so integrations of the same type appear next to
 * each other. Order is deterministic: by type, then by display name, then by
 * id as a final tie-breaker. Does not mutate the input array.
 */
export function sortConnectedIntegrationsByType<T extends ConnectedIntegrationLike>(integrations: T[]): T[] {
  return [...integrations].sort((a, b) => compareSortKeys(toSortKey(a), toSortKey(b)));
}
