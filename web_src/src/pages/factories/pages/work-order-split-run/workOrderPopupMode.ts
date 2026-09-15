export type WorkOrderPopupMode = "loading" | "classic" | "analysis";

function pinsAnalysisWithoutFlag(hasLookupIdentity: boolean, hasPlanningSession: boolean) {
  return !hasLookupIdentity || hasPlanningSession;
}

function refinementUsesAnalysis({
  hasAnalysisResult,
  analysisActive,
  isDraft,
  artifactsFailed,
}: {
  hasAnalysisResult: boolean;
  analysisActive: boolean;
  isDraft: boolean;
  artifactsFailed: boolean;
}) {
  return hasAnalysisResult || analysisActive || !isDraft || artifactsFailed;
}

function draftLookupPending(sessionLoading: boolean, artifactsLoading: boolean) {
  return sessionLoading || artifactsLoading;
}

export function workOrderPopupMode({
  hasPlanningSession,
  hasAnalysisResult = false,
  refinementEnabled,
  refinementLoading = false,
  sessionLoading = false,
  artifactsLoading = false,
  artifactsFailed = false,
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
  artifactsFailed?: boolean;
  analysisActive: boolean;
  hasLookupIdentity: boolean;
  isDraft?: boolean;
}): WorkOrderPopupMode {
  if (pinsAnalysisWithoutFlag(hasLookupIdentity, hasPlanningSession)) {
    return "analysis";
  }
  if (refinementLoading) {
    return "loading";
  }
  if (!refinementEnabled) {
    return "classic";
  }
  if (refinementUsesAnalysis({ hasAnalysisResult, analysisActive, isDraft, artifactsFailed })) {
    return "analysis";
  }
  return draftLookupPending(sessionLoading, artifactsLoading) ? "loading" : "classic";
}
