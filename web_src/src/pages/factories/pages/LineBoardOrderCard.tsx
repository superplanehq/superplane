import type { FactoriesWorkOrderCheckScore, FactoriesWorkOrderSummary } from "@/api-client";
import { useMemo, type ComponentProps } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  boardCardLoadsConfidenceChecks,
  clarityScoreFromChecks,
  confidenceScoreFromChecks,
} from "../lib/confidenceScore";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { draftCardAgentIsWorking, planningSessionHasPendingSurvey } from "./planningSessionView";
import { factoryShowsClarity, factoryShowsConfidence } from "./planningSettingsModel";

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
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [factory, order]);
  const showConfidence = boardCardLoadsConfidenceChecks(entry.displayStatus);
  const session = order.planningSession;
  const scores = cardScores(showConfidence, order.checkScores, session, isAnalyzing, {
    showClarity: factoryShowsClarity(factory),
    showConfidence: factoryShowsConfidence(factory),
  });

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      {...scores}
      hasAgentQuestion={showConfidence && planningSessionHasPendingSurvey(session)}
      onOpen={onOpen}
    />
  );
}

/**
 * Scores come from the list. The thinking state follows the planning session
 * summary on that list. A local backlog run still counts until the first score.
 * The opened card loads the full session.
 */
function cardScores(
  showConfidence: boolean,
  checks: FactoriesWorkOrderCheckScore[] | undefined,
  session: FactoriesWorkOrderSummary["planningSession"],
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
