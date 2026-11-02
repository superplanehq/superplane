import { PlanningReview } from "./PlanningReview";
import type { IntentAnalysisChat } from "./intentAnalysisChat";
import { hasPendingPlanningQuestions } from "./planningReviewState";

/** Plan controls stay in the conversation, above the composer. */
export function AnalysisPlanControls({
  analysis,
  chipsWorking,
  position,
}: {
  analysis: IntentAnalysisChat;
  chipsWorking: boolean;
  position: "conversation" | "composer";
}) {
  if (position !== "conversation") {
    return null;
  }
  if (!shouldShowPlanningReview(analysis)) {
    return null;
  }
  return (
    <PlanningReview
      open={Boolean(analysis.planPaneOpen)}
      title={analysis.planTitle}
      clarity={analysis.clarity}
      confidence={analysis.confidence}
      reviewMetrics={analysis.reviewMetrics}
      questionsPending={hasPendingPlanningQuestions(analysis.view)}
      showClarity={analysis.showClarity !== false}
      showConfidence={analysis.showConfidence !== false}
      isAnalyzing={chipsWorking}
      canTogglePlan={Boolean(analysis.canTogglePlan)}
      planStatus={analysis.planStatus}
      onToggle={analysis.onTogglePlan}
      creditVerdict={analysis.creditVerdict}
    />
  );
}

/** Keep credit blocks and finished reviews visible; a new run uses its live activity until a plan exists. */
function shouldShowPlanningReview(analysis: IntentAnalysisChat): boolean {
  if (analysis.creditVerdict || analysis.canTogglePlan) {
    return true;
  }
  switch (analysis.view.machineStatus) {
    case "queued":
    case "starting":
    case "running":
      return false;
    default:
      return true;
  }
}
