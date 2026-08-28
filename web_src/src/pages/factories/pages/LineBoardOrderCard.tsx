import type { FactoriesWorkOrder } from "@/api-client";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { useMemo } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { boardCardLoadsConfidenceChecks, confidenceScoreFromChecks } from "../lib/confidenceScore";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";

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
  const { data: checks = [] } = useWorkOrderChecks(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryId ?? "",
    order.id ?? "",
    { enabled: showConfidence },
  );

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      confidenceScore={showConfidence ? confidenceScoreFromChecks(checks) : undefined}
      isAnalyzing={showConfidence && isAnalyzing}
      onOpen={onOpen}
    />
  );
}
