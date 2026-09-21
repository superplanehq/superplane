import type { FactoriesWorkOrder, FactoriesWorkOrderCheck } from "@/api-client";
import { ANALYZING_WORK_ORDER_CHECKS_POLL_MS, useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { factoryPlanningEnabled, factoryShowsClarity, factoryShowsConfidence } from "./planningSettingsModel";
import { useEffect, useMemo, useRef, type ComponentProps } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  boardCardLoadsConfidenceChecks,
  clarityScoreFromChecks,
  confidenceScoreFromChecks,
} from "../lib/confidenceScore";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { draftCardAgentIsWorking, type PlanningSessionMachineInput } from "./planningSessionView";
import { useWorkOrderPlanningActivity } from "./useWorkOrderPlanningSurvey";

export function LineBoardOrderCard({
  order,
  workOrderCardContext,
  onOpenWorkOrder,
  isAnalyzing = false,
}: {
  order: FactoriesWorkOrder;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  isAnalyzing?: boolean;
}) {
  return (
    <LineBoardWorkOrderCard
      order={order}
      workOrderCardContext={workOrderCardContext}
      isAnalyzing={isAnalyzing}
      onOpen={() => {
        if (order.id) {
          onOpenWorkOrder(order.id, order);
        }
      }}
    />
  );
}

export function LineBoardWorkOrderCard({
  order,
  workOrderCardContext,
  onOpen,
  isAnalyzing = false,
}: {
  order: FactoriesWorkOrder;
  workOrderCardContext: WorkOrderCardContext;
  onOpen: () => void;
  isAnalyzing?: boolean;
}) {
  const { factory } = useFactoriesLayout();
  const refinementEnabled = factoryPlanningEnabled(factory);
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [factory, order]);
  const showConfidence = boardCardLoadsConfidenceChecks(entry.displayStatus);
  const isDraft = entry.displayStatus === "draft";
  const watchSession = refinementEnabled && isDraft && showConfidence;
  const session = useWorkOrderPlanningActivity(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    watchSession,
    isAnalyzing,
  );
  // Poll checks while anything may still write a score. The visible state
  // below is stricter and matches the refine strip.
  const showAnalysisActivity = watchSession && (session.isWorking || isAnalyzing);
  const { data: checks = [], refetch } = useWorkOrderChecks(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    {
      enabled: showConfidence,
      refetchInterval: showAnalysisActivity ? ANALYZING_WORK_ORDER_CHECKS_POLL_MS : false,
    },
  );
  const wasAnalyzing = useRef(showAnalysisActivity);
  useEffect(() => {
    if (wasAnalyzing.current && !showAnalysisActivity && showConfidence) {
      void refetch?.();
    }
    wasAnalyzing.current = showAnalysisActivity;
  }, [refetch, showAnalysisActivity, showConfidence]);
  const scores = cardScores(showConfidence, checks, session.session, isAnalyzing, {
    showClarity: factoryShowsClarity(factory),
    showConfidence: factoryShowsConfidence(factory),
  });

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      {...scores}
      hasAgentQuestion={session.hasAgentQuestion}
      onOpen={onOpen}
    />
  );
}

/** Scores from checks, and the strip rule for the thinking state. Nothing when the column hides scores. */
function cardScores(
  showConfidence: boolean,
  checks: FactoriesWorkOrderCheck[],
  session: PlanningSessionMachineInput | null,
  backlogAnalyzing: boolean,
  visibility: { showClarity: boolean; showConfidence: boolean },
): Pick<
  ComponentProps<typeof WorkOrderCard>,
  "clarityScore" | "confidenceScore" | "isAnalyzing" | "showClarity" | "showConfidenceScore"
> {
  if (!showConfidence) {
    return { isAnalyzing: false, showClarity: false, showConfidenceScore: false };
  }
  const clarityScore = visibility.showClarity ? clarityScoreFromChecks(checks) : undefined;
  const confidenceScore = visibility.showConfidence ? confidenceScoreFromChecks(checks) : undefined;
  return {
    clarityScore,
    confidenceScore,
    showClarity: visibility.showClarity,
    showConfidenceScore: visibility.showConfidence,
    isAnalyzing: draftCardAgentIsWorking(session, backlogAnalyzing, clarityScore ?? confidenceScore),
  };
}
