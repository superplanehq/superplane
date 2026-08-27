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
}: {
  order: FactoriesWorkOrder;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
}) {
  return (
    <LineBoardWorkOrderCard
      order={order}
      workOrderCardContext={workOrderCardContext}
      onOpen={() => {
        if (order.id) {
          onOpenWorkOrder(order.id);
        }
      }}
    />
  );
}

export function LineBoardWorkOrderCard({
  order,
  workOrderCardContext,
  onOpen,
}: {
  order: FactoriesWorkOrder;
  workOrderCardContext: WorkOrderCardContext;
  onOpen: () => void;
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
      onOpen={onOpen}
    />
  );
}
