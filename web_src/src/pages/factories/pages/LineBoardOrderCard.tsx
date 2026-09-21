import type { FactoriesWorkOrderCheckScore, FactoriesWorkOrderSummary } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
import { useMemo, type ComponentProps } from "react";

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
  order: FactoriesWorkOrderSummary;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrderSummary) => void;
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
  order: FactoriesWorkOrderSummary;
  workOrderCardContext: WorkOrderCardContext;
  onOpen: () => void;
  isAnalyzing?: boolean;
}) {
  const { factory } = useFactoriesLayout();
  const refinementEnabled = useExperimentalFeature(workOrderCardContext.organizationId).has(
    FEATURE_FACTORY_CREATE_WITH_AGENT,
  );
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
  const scores = cardScores(showConfidence, order.checkScores, session.session, isAnalyzing);

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
  checks: FactoriesWorkOrderCheckScore[] | undefined,
  session: PlanningSessionMachineInput | null,
  backlogAnalyzing: boolean,
): Pick<ComponentProps<typeof WorkOrderCard>, "clarityScore" | "confidenceScore" | "isAnalyzing"> {
  if (!showConfidence) {
    return { isAnalyzing: false };
  }
  const clarityScore = clarityScoreFromChecks(checks);
  const confidenceScore = confidenceScoreFromChecks(checks);
  return {
    clarityScore,
    confidenceScore,
    isAnalyzing: draftCardAgentIsWorking(session, backlogAnalyzing, clarityScore ?? confidenceScore),
  };
}
