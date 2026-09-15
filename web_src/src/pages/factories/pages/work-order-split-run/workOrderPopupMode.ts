export type WorkOrderPopupMode = "loading" | "classic" | "analysis";

export function workOrderPopupMode({
  hasPlanningSession,
  hasAnalysisResult = false,
  refinementEnabled,
  refinementLoading = false,
  sessionLoading = false,
  artifactsLoading = false,
  analysisActive,
  hasLookupIdentity,
  isDraft = true,
}: {
  hasPlanningSession: boolean;
  hasAnalysisResult?: boolean;
  refinementEnabled: boolean;
  refinementLoading?: boolean;
  sessionLoading?: boolean;
  artifactsLoading?: boolean;
  analysisActive: boolean;
  hasLookupIdentity: boolean;
  isDraft?: boolean;
}): WorkOrderPopupMode {
  if (!hasLookupIdentity || hasPlanningSession) {
    return "analysis";
  }
  if (refinementLoading) {
    return "loading";
  }
  if (!refinementEnabled) {
    return "classic";
  }
  if (hasAnalysisResult || analysisActive || !isDraft) {
    return "analysis";
  }
  return sessionLoading || artifactsLoading ? "loading" : "classic";
}
