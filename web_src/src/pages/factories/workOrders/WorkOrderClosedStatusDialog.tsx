import { useMemo, useState } from "react";
import { Search } from "lucide-react";

import type { FactoriesWorkOrderSummary } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useFactoryWorkOrdersPage } from "@/hooks/useFactoryData";
import { Link } from "react-router";

import { workOrderOpenPath } from "../lib/factoryPagePaths";
import { BOARD_DONE_PAGE_SIZE, BOARD_DONE_STATES } from "../lib/workOrderListPagination";
import { getWorkOrderDisplayKey, getWorkOrderDisplayStatus } from "../lib/workOrderProgress";
import {
  CLOSED_STATUS_DIALOG_COPY,
  CLOSED_STATUS_DIALOG_RESULTS,
  SEND_WORK_ORDER_TO_BACKLOG_COPY,
  closedStatusTaskMatchesSearch,
} from "../lib/sendWorkOrderToBacklog";
import { SendWorkOrderToBacklogForm } from "./SendWorkOrderToBacklogForm";
import { WorkOrderStatusBadge } from "./WorkOrderStatusIcon";

export interface WorkOrderClosedStatusDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  canManage?: boolean;
  onOpenChange: (open: boolean) => void;
}

export function WorkOrderClosedStatusDialog({
  open,
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  canManage = true,
  onOpenChange,
}: WorkOrderClosedStatusDialogProps) {
  const [search, setSearch] = useState("");
  const page = useFactoryWorkOrdersPage(organizationId, factoryId, BOARD_DONE_STATES, BOARD_DONE_PAGE_SIZE, {
    results: [...CLOSED_STATUS_DIALOG_RESULTS],
    lineId,
  });
  const visibleOrders = useMemo(
    () => page.orders.filter((order) => closedStatusTaskMatchesSearch(order, factoryKey, search)),
    [factoryKey, page.orders, search],
  );
  const searchActive = search.trim().length > 0;
  const showSearch = !page.isLoading && (page.orders.length > 0 || searchActive);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          setSearch("");
        }
        onOpenChange(nextOpen);
      }}
    >
      <DialogContent className="flex max-h-[80vh] flex-col sm:max-w-lg" data-testid="work-order-closed-status-dialog">
        <DialogHeader>
          <DialogTitle>{CLOSED_STATUS_DIALOG_COPY.title}</DialogTitle>
          <DialogDescription>{CLOSED_STATUS_DIALOG_COPY.description}</DialogDescription>
        </DialogHeader>
        {showSearch ? (
          <div className="relative">
            <Search
              className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
              aria-hidden
            />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={CLOSED_STATUS_DIALOG_COPY.searchPlaceholder}
              aria-label={CLOSED_STATUS_DIALOG_COPY.searchPlaceholder}
              className="h-8 pl-8 text-[13px]"
              data-testid="work-order-closed-status-search"
            />
          </div>
        ) : null}
        <ClosedStatusTaskList
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          canManage={canManage}
          emptyLabel={searchActive ? CLOSED_STATUS_DIALOG_COPY.searchEmpty : CLOSED_STATUS_DIALOG_COPY.empty}
          orders={visibleOrders}
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
  if (orders.length === 0 && !hasNextPage) {
    return <p className="text-[13px] text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <div className="flex min-h-0 flex-col gap-3">
      {orders.length === 0 ? <p className="text-[13px] text-muted-foreground">{emptyLabel}</p> : null}
      {orders.length > 0 ? (
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
      ) : null}
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
  const displayStatus = getWorkOrderDisplayStatus(order);

  return (
    <div className="flex flex-col gap-3 px-3 py-3" data-testid={`work-order-closed-status-row-${orderId}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <Link to={href} className="min-w-0 truncate text-[13px] font-medium text-foreground hover:underline">
              {order.title?.trim() || "Untitled task"}
            </Link>
            <WorkOrderStatusBadge status={displayStatus} />
          </div>
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
