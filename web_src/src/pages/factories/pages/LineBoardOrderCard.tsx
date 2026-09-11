import type { FactoriesWorkOrder } from "@/api-client";
import { ANALYZING_WORK_ORDER_CHECKS_POLL_MS, useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
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
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [factory, order]);
  const showConfidence = boardCardLoadsConfidenceChecks(entry.displayStatus);
  const { data: checks = [], refetch } = useWorkOrderChecks(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    {
      enabled: showConfidence,
      refetchInterval: isAnalyzing ? ANALYZING_WORK_ORDER_CHECKS_POLL_MS : false,
    },
  );
  const wasAnalyzing = useRef(isAnalyzing);
  useEffect(() => {
    if (wasAnalyzing.current && !isAnalyzing && showConfidence) {
      void refetch?.();
    }
    wasAnalyzing.current = isAnalyzing;
  }, [isAnalyzing, refetch, showConfidence]);
  const isDraft = entry.displayStatus === "draft";
  const hasAgentQuestion = useWorkOrderPlanningSurvey(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    isDraft && isAnalyzing,
  );

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      confidenceScore={showConfidence ? confidenceScoreFromChecks(checks) : undefined}
      isAnalyzing={showConfidence && isAnalyzing}
      hasAgentQuestion={isDraft && hasAgentQuestion}
      onOpen={onOpen}
    />
  );
}
