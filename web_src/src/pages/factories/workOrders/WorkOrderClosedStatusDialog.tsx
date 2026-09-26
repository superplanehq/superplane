import { useState } from "react";

import type { FactoriesWorkOrderSummary } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useFactoryWorkOrdersPage } from "@/hooks/useFactoryData";
import { Link } from "react-router";

import { workOrderOpenPath } from "../lib/factoryPagePaths";
import { BOARD_DONE_PAGE_SIZE, BOARD_DONE_STATES } from "../lib/workOrderListPagination";
import { getWorkOrderDisplayKey } from "../lib/workOrderProgress";
import {
  CLOSED_STATUS_DIALOG_COPY,
  SEND_WORK_ORDER_TO_BACKLOG_COPY,
  closedWorkOrderResultForDialogStatus,
} from "../lib/sendWorkOrderToBacklog";
import { SendWorkOrderToBacklogForm } from "./SendWorkOrderToBacklogForm";

export interface WorkOrderClosedStatusDialogProps {
  open: boolean;
  status: "failed" | "rejected";
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  canManage?: boolean;
  onOpenChange: (open: boolean) => void;
}

export function WorkOrderClosedStatusDialog({
  open,
  status,
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  canManage = true,
  onOpenChange,
}: WorkOrderClosedStatusDialogProps) {
  const copy = CLOSED_STATUS_DIALOG_COPY[status];
  const page = useFactoryWorkOrdersPage(organizationId, factoryId, BOARD_DONE_STATES, BOARD_DONE_PAGE_SIZE, {
    results: [closedWorkOrderResultForDialogStatus(status)],
    lineId,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[80vh] flex-col sm:max-w-lg" data-testid={`work-order-${status}-dialog`}>
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>{copy.description}</DialogDescription>
        </DialogHeader>
        <ClosedStatusTaskList
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          canManage={canManage}
          emptyLabel={copy.empty}
          orders={page.orders}
          isLoading={page.isLoading}
          hasNextPage={page.hasNextPage}
          isFetchingNextPage={page.isFetchingNextPage}
          onLoadMore={() => {
            void page.fetchNextPage();
          }}
        />
      </DialogContent>
    </Dialog>
  );
}

function ClosedStatusTaskList({
  organizationId,
  factoryId,
  factoryKey,
  canManage,
  emptyLabel,
  orders,
  isLoading,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canManage: boolean;
  emptyLabel: string;
  orders: FactoriesWorkOrderSummary[];
  isLoading: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading tasks…</p>;
  }
  if (orders.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <div className="flex min-h-0 flex-col gap-3">
      <ul className="divide-y divide-border overflow-y-auto rounded-md border border-border">
        {orders.map((order) => (
          <li key={order.id ?? order.number}>
            <ClosedStatusTaskRow
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              canManage={canManage}
              order={order}
            />
          </li>
        ))}
      </ul>
      {hasNextPage ? (
        <Button type="button" variant="outline" size="sm" disabled={isFetchingNextPage} onClick={onLoadMore}>
          Load more
        </Button>
      ) : null}
    </div>
  );
}

function ClosedStatusTaskRow({
  organizationId,
  factoryId,
  factoryKey,
  canManage,
  order,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canManage: boolean;
  order: FactoriesWorkOrderSummary;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const orderId = order.id ?? "";
  const href = workOrderOpenPath(organizationId, factoryKey, order.number);
  const identifier = getWorkOrderDisplayKey(order, factoryKey);

  return (
    <div className="flex flex-col gap-3 px-3 py-3" data-testid={`work-order-closed-status-row-${orderId}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <Link to={href} className="block truncate text-[13px] font-medium text-foreground hover:underline">
            {order.title?.trim() || "Untitled task"}
          </Link>
          <p className="text-[11px] text-muted-foreground">{identifier}</p>
        </div>
        {canManage && orderId && !confirmOpen ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setConfirmOpen(true)}
            data-testid={`send-to-backlog-${orderId}`}
          >
            {SEND_WORK_ORDER_TO_BACKLOG_COPY.action}
          </Button>
        ) : null}
      </div>
      {confirmOpen && orderId ? (
        <SendWorkOrderToBacklogForm
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          pullRequests={order.pullRequests}
          canSubmit={canManage}
          onSent={() => setConfirmOpen(false)}
        />
      ) : null}
    </div>
  );
}
