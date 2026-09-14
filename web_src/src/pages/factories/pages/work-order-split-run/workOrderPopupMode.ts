export type WorkOrderPopupMode = "classic" | "analysis";

export function workOrderPopupMode({
  hasPlanningSession,
  refinementEnabled,
  hasLookupIdentity,
}: {
  hasPlanningSession: boolean;
  refinementEnabled: boolean;
  hasLookupIdentity: boolean;
}): WorkOrderPopupMode {
  if (!hasLookupIdentity || hasPlanningSession || refinementEnabled) {
    return "analysis";
  }
  return "classic";
}
