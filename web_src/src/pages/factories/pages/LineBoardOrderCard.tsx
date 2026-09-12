import type { FactoriesWorkOrder } from "@/api-client";
import { ANALYZING_WORK_ORDER_CHECKS_POLL_MS, useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
import { useEffect, useMemo, useRef } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { boardCardLoadsConfidenceChecks, confidenceScoreFromChecks } from "../lib/confidenceScore";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { useWorkOrderPlanningSurvey } from "./useWorkOrderPlanningSurvey";

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
  const refinementEnabled = useExperimentalFeature(workOrderCardContext.organizationId).has(
    FEATURE_FACTORY_CREATE_WITH_AGENT,
  );
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [factory, order]);
  const showConfidence = boardCardLoadsConfidenceChecks(entry.displayStatus);
  const isDraft = entry.displayStatus === "draft";
  const showAnalysisActivity = refinementEnabled && isDraft && Boolean(isAnalyzing);
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
  const hasAgentQuestion = useWorkOrderPlanningSurvey(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    showAnalysisActivity,
  );

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      confidenceScore={showConfidence ? confidenceScoreFromChecks(checks) : undefined}
      isAnalyzing={showConfidence && showAnalysisActivity}
      hasAgentQuestion={showAnalysisActivity && hasAgentQuestion}
      onOpen={onOpen}
    />
  );
}
