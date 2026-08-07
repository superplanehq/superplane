export function factoryListPath(organizationId: string) {
  return `/${organizationId}/factories`;
}

export function factoryDetailPath(organizationId: string, factoryId: string) {
  return `${factoryListPath(organizationId)}/${factoryId}`;
}

export function factoryOverviewPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/overview`;
}

export function workOrdersPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/work-orders`;
}

export function createWorkOrderPath(organizationId: string, factoryId: string) {
  return `${workOrdersPath(organizationId, factoryId)}/new`;
}

export function workOrderDetailPath(organizationId: string, factoryId: string, orderId: string) {
  return `${workOrdersPath(organizationId, factoryId)}/${orderId}`;
}

export function automationsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/automations`;
}

export function createFactoryLinePath(organizationId: string, factoryId: string) {
  return `${automationsPath(organizationId, factoryId)}/new`;
}

export function factoryLineDetailPath(organizationId: string, factoryId: string, lineId: string) {
  return `${automationsPath(organizationId, factoryId)}/${lineId}`;
}

export function editFactoryLinePath(organizationId: string, factoryId: string, lineId: string) {
  return `${automationsPath(organizationId, factoryId)}/${lineId}/edit`;
}

export function factorySettingsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/settings`;
}

export function factorySettingsSectionPath(organizationId: string, factoryId: string, section: string) {
  return `${factorySettingsPath(organizationId, factoryId)}/${section}`;
}

export function factoryMissionsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/missions`;
}

export function factoryWikiPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/wiki`;
}

export function factoryVelocityPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/velocity`;
}
