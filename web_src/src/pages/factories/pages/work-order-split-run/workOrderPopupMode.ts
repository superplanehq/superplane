export type WorkOrderPopupMode = "classic" | "analysis";

export function workOrderPopupMode({
  hasPlanningSession,
  hasAnalysisResult = false,
  refinementEnabled,
  analysisActive,
  hasLookupIdentity,
}: {
  hasPlanningSession: boolean;
  hasAnalysisResult?: boolean;
  refinementEnabled: boolean;
  analysisActive: boolean;
  hasLookupIdentity: boolean;
}): WorkOrderPopupMode {
  if (!hasLookupIdentity || hasPlanningSession) {
    return "analysis";
  }
  if (!refinementEnabled) {
    return "classic";
  }
  return hasAnalysisResult || analysisActive ? "analysis" : "classic";
}
