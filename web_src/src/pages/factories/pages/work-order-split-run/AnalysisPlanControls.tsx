import { ComposerPlanStack } from "./ComposerPlanControls";
import { PlanningReview } from "./PlanningReview";
import type { IntentAnalysisChat } from "./intentAnalysisChat";

/** Keep the original controls above the composer until the organization opts in. */
export function AnalysisPlanControls({
  analysis,
  chipsWorking,
  position,
}: {
  analysis: IntentAnalysisChat;
  chipsWorking: boolean;
  position: "conversation" | "composer";
}) {
  const inConversation = Boolean(analysis.planningReviewEnabled);
  if (inConversation !== (position === "conversation")) {
    return null;
  }
  const props = {
    open: Boolean(analysis.planPaneOpen),
    clarity: analysis.clarity,
    confidence: analysis.confidence,
    showClarity: analysis.showClarity !== false,
    showConfidence: analysis.showConfidence !== false,
    isAnalyzing: chipsWorking,
    canTogglePlan: Boolean(analysis.canTogglePlan),
    planStatus: analysis.planStatus,
    onToggle: analysis.onTogglePlan,
    creditVerdict: analysis.creditVerdict,
  };
  if (inConversation) {
    return <PlanningReview {...props} title={analysis.planTitle} reviewMetrics={analysis.reviewMetrics} />;
  }
  return <ComposerPlanStack {...props} actions={analysis.closedDecision} modelSelect={analysis.modelSelect} />;
}
