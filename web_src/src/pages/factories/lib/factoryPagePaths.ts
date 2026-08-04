export function factoryListPath(organizationId: string) {
  return `/${organizationId}/factories`;
}

export function factoryDetailPath(organizationId: string, factoryId: string) {
  return `${factoryListPath(organizationId)}/${factoryId}`;
}

export function createWorkOrderPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/orders/new`;
}
