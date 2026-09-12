export type WorkOrderPopupMode = "classic" | "analysis";

export function workOrderPopupMode({
  hasPlanningSession,
  refinementEnabled,
  analysisActive,
  hasLookupIdentity,
}: {
  hasPlanningSession: boolean;
  refinementEnabled: boolean;
  analysisActive: boolean;
  hasLookupIdentity: boolean;
}): WorkOrderPopupMode {
  if (!hasLookupIdentity || hasPlanningSession) {
    return "analysis";
  }
  return refinementEnabled && analysisActive ? "analysis" : "classic";
}
