import { PlanningReview } from "./PlanningReview";
import type { IntentAnalysisChat } from "./intentAnalysisChat";

export function AnalysisPlanReview({
  analysis,
  chipsWorking,
}: {
  analysis: IntentAnalysisChat;
  chipsWorking: boolean;
}) {
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
  return <PlanningReview {...props} title={analysis.planTitle} reviewMetrics={analysis.reviewMetrics} />;
}
