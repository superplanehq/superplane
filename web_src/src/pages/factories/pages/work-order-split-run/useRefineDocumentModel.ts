import { useState, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { analysisPlanBody, hasAnalysisPlan } from "../../lib/analysisOutcome";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import type { ComposerScore } from "./ComposerPlanControls";
import { latestPlanScore } from "./latestPlanScore";
import { usePlanChipStatus } from "./planChipStatus";
import { useRefineLayoutPreference } from "./refineLayoutPreference";
import { splitRunIntentDocument } from "./splitRunPopupModel";
import type { IntentAnalysisChat } from "./WorkOrderIntentRequest";
import {
  DEFAULT_INTENT_LEFT_PERCENT,
  DEFAULT_REFINE_INTENT_LEFT_PERCENT,
  useSplitRunPanePercent,
} from "./useSplitRunPanePercent";

const SESSION_TITLE_FALLBACK = "Task";

/** Clarity prefers the live plan message; the check is the durable fallback. */
function refineClarity(analysis: IntentAnalysisChat | undefined, clarity?: WorkOrderCheckPresentation): ComposerScore {
  const liveScore = analysis ? latestPlanScore(analysis.view.messages) : undefined;
  return {
    score: liveScore ?? clarity?.score,
    summary: clarity?.summary?.trim(),
  };
}

function refineConfidence(confidence?: WorkOrderCheckPresentation): ComposerScore {
  return {
    score: confidence?.score,
    summary: confidence?.summary?.trim(),
  };
}

function refinePaneLayout(refineOpen: boolean, planPaneOpen: boolean, percent: number) {
  const showPlanPane = !refineOpen || planPaneOpen;
  return {
    showPlanPane,
    chatSolo: refineOpen && !planPaneOpen,
    mountPlanPane: refineOpen || showPlanPane,
    planWidth: showPlanPane ? `${100 - percent}%` : "0%",
    chatWidth: showPlanPane ? `${percent}%` : "100%",
  };
}

export function useRefineDocumentModel({
  title,
  description,
  artifacts,
  clarity,
  confidence,
  isAnalyzing,
  resultFooter,
  analysis,
  contextSidebar,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  clarity?: WorkOrderCheckPresentation;
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
  contextSidebar?: ReactNode;
}) {
  const refineOpen = Boolean(analysis) && !contextSidebar;
  const [planOpenedHere, setPlanOpenedHere] = useState(false);
  const layout = useRefineLayoutPreference();
  const hasPlan = hasAnalysisPlan(artifacts);
  const planPaneOpen = layout.planOpen && hasPlan;
  const split = useSplitRunPanePercent({
    defaultPercent: refineOpen ? DEFAULT_REFINE_INTENT_LEFT_PERCENT : DEFAULT_INTENT_LEFT_PERCENT,
    minPercent: 28,
    maxPercent: 68,
  });
  const panes = refinePaneLayout(refineOpen, planPaneOpen, split.percent);
  const planStatus = usePlanChipStatus(analysisPlanBody(artifacts), planOpenedHere && planPaneOpen);

  return {
    refineOpen,
    split,
    document: splitRunIntentDocument({ artifacts, description }),
    sessionTitle: title.trim() || SESSION_TITLE_FALLBACK,
    ...panes,
    showClosedDecision: refineOpen && resultFooter != null,
    analysisChat: bindRefineChat({
      analysis,
      planPaneOpen,
      hasPlan,
      layout,
      planStatus,
      clarity: refineClarity(analysis, clarity),
      confidence: refineConfidence(confidence),
      isAnalyzing,
      onOpenPlan: () => setPlanOpenedHere(true),
    }),
  };
}

function bindRefineChat({
  analysis,
  planPaneOpen,
  hasPlan,
  layout,
  planStatus,
  clarity,
  confidence,
  isAnalyzing,
  onOpenPlan,
}: {
  analysis?: IntentAnalysisChat;
  planPaneOpen: boolean;
  hasPlan: boolean;
  layout: ReturnType<typeof useRefineLayoutPreference>;
  planStatus: ReturnType<typeof usePlanChipStatus>;
  clarity: ComposerScore;
  confidence: ComposerScore;
  isAnalyzing: boolean;
  onOpenPlan: () => void;
}): IntentAnalysisChat | undefined {
  if (!analysis) {
    return undefined;
  }
  return {
    ...analysis,
    planPaneOpen,
    onTogglePlan: () => {
      onOpenPlan();
      layout.togglePlan();
    },
    canTogglePlan: hasPlan,
    openSummary: layout.openSummary,
    onToggleSummary: layout.toggleSummary,
    clarity,
    confidence,
    planStatus,
    isAnalyzing,
  };
}
