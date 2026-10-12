import { PlanningReview } from "./PlanningReview";
import type { IntentAnalysisChat } from "./intentAnalysisChat";

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
  return (
    <PlanningReview
      open={Boolean(analysis.planPaneOpen)}
      title={analysis.planTitle}
      clarity={analysis.clarity}
      confidence={analysis.confidence}
      reviewMetrics={analysis.reviewMetrics}
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
