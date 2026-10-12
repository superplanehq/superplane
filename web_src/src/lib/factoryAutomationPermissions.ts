export function resolveCanUpdateFactory(
  permissionsLoading: boolean,
  canAct: (resource: string, action: string) => boolean,
) {
  return !permissionsLoading && canAct("factories", "update");
}

export function resolveCanUpdateFactoryAutomation(
  permissionsLoading: boolean,
  canAct: (resource: string, action: string) => boolean,
) {
  return !permissionsLoading && canAct("canvases", "update");
}
