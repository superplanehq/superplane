import type { FactoriesWorkOrder } from "@/api-client";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { usePermissions } from "@/contexts/usePermissions";
import { factoryBoardLaneScrollKey, useFactoryBoardLaneScroll } from "@/hooks/useFactoryBoardLaneScroll";
import { type RefreshBacklogResult, useFactoryIntakes, useRefreshBacklog } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";
import { Loader2, Search } from "lucide-react";
import { type KeyboardEvent } from "react";

import { WorkOrderBoardLane, workOrderKanbanLaneScrollClassName } from "../workOrders/WorkOrderBoardChrome";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { BacklogCreatePopover } from "./BacklogCreatePopover";
import { BacklogIntakeSources } from "./BacklogIntakeSources";
import { BacklogSettingsDialog } from "./BacklogSettingsDialog";
import { columnAutomationRowsSubheader } from "./columnAutomationRowsSubheader";
import { ColumnAutomationsHeaderSlot } from "./ColumnAutomationsIndicator";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import type { ColumnAutomation } from "../lib/columnAutomations";
import type { ColumnAutomationRowAction } from "./ColumnAutomationsPopup";
import { LineBoardColumnCardList, LineBoardOrderCard } from "./LineBoardOrderCard";
import type { LineBoardColumnColorView } from "../lib/lineBoardColumnColorViewPreference";
import { lineBoardColumnLaneProps, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { isFirstRunOnboardingFactory, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { BacklogOnboardingCard } from "./onboarding/first-run/BacklogOnboardingCard";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";
import { BACKLOG_REFRESH_COPY, backlogRefreshToast, canRefreshBacklog } from "./backlogRefresh";
import { BACKLOG_COLUMN_SEARCH_COPY } from "../lib/backlogColumnSearch";
import { useBacklogColumnSearch, type BacklogColumnPaging } from "./useBacklogColumnSearch";

export type BacklogColumnProps = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  orders: FactoriesWorkOrder[];
  title: string;
  size: number | null;
  settingsOpen: boolean;
  onOpenSettings: () => void;
  onCloseSettings: () => void;
  onSaveSettings: (settings: { name: string; size: number | null }) => void;
  colorId: LineBoardColumnColorId | null;
  colorView?: LineBoardColumnColorView;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onRename: (title: string) => void;
  onCreateWorkOrder: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  /** Tasks the Backlog automation analyzes right now. */
  analyzingOrderIds?: ReadonlySet<string>;
  /** Short credit label for a draft whose analysis stopped for hosted credit. */
  creditFailureLabels?: ReadonlyMap<string, string>;
  /** Intakes that open tasks in this backlog, listed at its head. */
  intakePanel?: BacklogIntakePanel;
  /** Opens the Add intake picker from the overflow menu. Hidden when unset. */
  onAddIntake?: () => void;
  /** Column automations for the header icons. Hidden when unset. */
  automations?: ColumnAutomation[];
  /** Rows the automation subheader reserves. Shared across the board. Hidden when unset. */
  automationRowCount?: number;
  onAutomationRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  paging?: BacklogColumnPaging;
  cardsPending?: boolean;
};

export type BacklogIntakePanel = {
  sources: ConfiguredLineIntakeSource[];
  /** Show Add intake. Hidden on the board until the flow is ready. */
  showAddIntake: boolean;
  onOpenSettings: (intake: ConfiguredLineIntakeSource) => void;
  onAddIntake: () => void;
};

export function BacklogColumn({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  orders,
  title,
  size,
  settingsOpen,
  onOpenSettings,
  onCloseSettings,
  onSaveSettings,
  colorId,
  colorView,
  onColorChange,
  canCreateWorkOrder,
  canRename,
  onRename,
  onCreateWorkOrder,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  creditFailureLabels,
  intakePanel,
  onAddIntake,
  automations,
  automationRowCount,
  onAutomationRowAction,
  paging,
  cardsPending = false,
}: BacklogColumnProps) {
  const lane = lineBoardColumnLaneProps(colorId, colorView, { mutedFallback: true });
  const search = useBacklogColumnSearch(orders, paging);
  const atCapacity = size != null && orders.length >= size;
  const canAdd = canCreateWorkOrder && !atCapacity;
  const createMenu = useBacklogCreateMenu(organizationId, factoryId, onOpenWorkOrder);
  const { canAct } = usePermissions();
  const canUpdateWorkOrders = canAct("work_orders", "update");
  const intakesQuery = useFactoryIntakes(organizationId, factoryId);
  const refreshBacklog = useRefreshBacklog(organizationId, factoryId);
  const createPopover = backlogCreatePopoverProps({
    canAdd,
    atCapacity,
    createMenu,
    onCreateWorkOrder,
  });
  const refreshBacklogAction = canRefreshBacklog(intakesQuery.data, canUpdateWorkOrders)
    ? () => {
        void runBacklogRefresh(refreshBacklog.mutateAsync);
      }
    : undefined;

  return (
    <>
      <WorkOrderBoardLane
        title={title}
        label={title}
        canRename={canRename}
        onRename={onRename}
        titleTestId="lines-column-title-backlog"
        count={search.columnCount}
        tone="neutral"
        surfaceClassName={lane.surfaceClassName}
        emptyDescription="No tasks in the backlog."
        emptyContent={backlogEmptyContent(factoryKey, search.searchActive)}
        keepChildrenWhenEmpty
        className={lane.className}
        actions={
          <BacklogColumnHeaderActions
            title={title}
            createPopover={createPopover}
            automations={automations}
            automationRowCount={automationRowCount}
            onAutomationRowAction={onAutomationRowAction}
            onOpenSettings={onOpenSettings}
            onAddIntake={onAddIntake}
            onRefreshBacklog={refreshBacklogAction}
            refreshBacklogPending={refreshBacklog.isPending}
            colorId={colorId}
            onColorChange={onColorChange}
          />
        }
        subheader={columnAutomationRowsSubheader({
          title,
          automations,
          rowCount: automationRowCount,
          onRowAction: onAutomationRowAction,
          testId: "lines-backlog-automation-rows",
        })}
        banner={<BacklogColumnSearchBanner query={search.query} onQueryChange={search.setQuery} panel={intakePanel} />}
        testId="lines-backlog-column"
      >
        <BacklogColumnOrderList
          orders={search.visibleOrders}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={analyzingOrderIds}
          creditFailureLabels={creditFailureLabels}
          atCapacity={atCapacity}
          search={search}
          createPopover={createPopover}
          paging={paging}
          cardsPending={cardsPending}
          scrollPersistenceKey={lineId ? factoryBoardLaneScrollKey(factoryKey, lineId, "backlog") : undefined}
        />
      </WorkOrderBoardLane>
      <BacklogSettingsDialog
        open={settingsOpen}
        name={title}
        size={size}
        onSave={onSaveSettings}
        onClose={onCloseSettings}
      />
    </>
  );
}

function backlogEmptyContent(factoryKey: string, searchActive: boolean) {
  if (searchActive || !isFirstRunOnboardingFactory(factoryKey)) {
    return undefined;
  }
  return <BacklogOnboardingCard />;
}

function BacklogColumnSearchBanner({
  query,
  onQueryChange,
  panel,
}: {
  query: string;
  onQueryChange: (query: string) => void;
  panel?: BacklogIntakePanel;
}) {
  return (
    <div className="flex shrink-0 flex-col gap-2 pb-2">
      <BacklogColumnSearch query={query} onQueryChange={onQueryChange} />
      <BacklogColumnBanner panel={panel} />
    </div>
  );
}

function BacklogColumnSearch({ query, onQueryChange }: { query: string; onQueryChange: (query: string) => void }) {
  const clearQuery = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== "Escape") {
      return;
    }
    event.preventDefault();
    onQueryChange("");
  };

  return (
    <div className="relative">
      <Search
        className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
        aria-hidden
      />
      <Input
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        onKeyDown={clearQuery}
        placeholder={BACKLOG_COLUMN_SEARCH_COPY.placeholder}
        aria-label={BACKLOG_COLUMN_SEARCH_COPY.placeholder}
        className="h-8 w-full bg-background pl-8 text-[13px] shadow-none"
        data-testid="lines-backlog-search"
      />
    </div>
  );
}

function BacklogColumnHeaderActions({
  title,
  createPopover,
  automations,
  automationRowCount,
  onAutomationRowAction,
  onOpenSettings,
  onAddIntake,
  onRefreshBacklog,
  refreshBacklogPending,
  colorId,
  onColorChange,
}: Pick<
  BacklogColumnProps,
  | "title"
  | "automations"
  | "automationRowCount"
  | "onAutomationRowAction"
  | "onOpenSettings"
  | "onAddIntake"
  | "colorId"
  | "onColorChange"
> & {
  createPopover: BacklogCreatePopoverProps;
  onRefreshBacklog?: () => void;
  refreshBacklogPending?: boolean;
}) {
  return (
    <div className="flex shrink-0 items-center gap-0.5">
      {automationRowCount ? null : (
        <ColumnAutomationsHeaderSlot
          title={title}
          automations={automations}
          onRowAction={onAutomationRowAction}
          testId="lines-backlog-automations"
        />
      )}
      <BacklogCreatePopover {...createPopover} />
      <ColumnLaneMenu
        title={title}
        testId="lines-backlog-menu"
        onEdit={onOpenSettings}
        onAddIntake={onAddIntake}
        onRefreshBacklog={onRefreshBacklog}
        refreshBacklogPending={refreshBacklogPending}
        colorId={colorId}
        onColorChange={onColorChange}
      />
    </div>
  );
}

function BacklogColumnBanner({ panel }: { panel?: BacklogIntakePanel }) {
  if (!panel) {
    return null;
  }

  return (
    <BacklogIntakeSources
      intakes={panel.sources}
      showAddIntake={panel.showAddIntake}
      onOpenSettings={panel.onOpenSettings}
      onAddIntake={panel.onAddIntake}
    />
  );
}

function BacklogColumnOrderList({
  orders,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  creditFailureLabels,
  atCapacity,
  search,
  createPopover,
  paging,
  cardsPending,
  scrollPersistenceKey,
}: Pick<
  BacklogColumnProps,
  | "orders"
  | "workOrderCardContext"
  | "onOpenWorkOrder"
  | "analyzingOrderIds"
  | "creditFailureLabels"
  | "paging"
  | "cardsPending"
> & {
  atCapacity: boolean;
  search: ReturnType<typeof useBacklogColumnSearch>;
  createPopover: BacklogCreatePopoverProps;
  scrollPersistenceKey?: string;
}) {
  const { scrollRef, handleScroll } = useFactoryBoardLaneScroll(scrollPersistenceKey, !cardsPending);
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: search.searchActive ? false : paging?.hasMore,
    isLoading: paging?.isLoading,
    onLoadMore: paging?.onLoadMore,
  });

  return (
    <LineBoardColumnCardList
      ref={scrollRef}
      pending={Boolean(cardsPending)}
      className={workOrderKanbanLaneScrollClassName}
      testId="lines-backlog-column-scroll"
      onScroll={(element) => {
        handleScroll(element);
        loadMoreIfNeeded(element);
      }}
    >
      {orders.map((order) => (
        <li key={order.id}>
          <LineBoardOrderCard
            order={order}
            workOrderCardContext={workOrderCardContext}
            onOpenWorkOrder={onOpenWorkOrder}
            isAnalyzing={Boolean(order.id && analyzingOrderIds?.has(order.id))}
            creditLabel={order.id ? creditFailureLabels?.get(order.id) : undefined}
          />
        </li>
      ))}
      {search.showNoMatch ? (
        <li>
          <p className="rounded-md border border-dashed border-border/60 px-3 py-6 text-center text-[12px] text-muted-foreground">
            {BACKLOG_COLUMN_SEARCH_COPY.noMatch}
          </p>
        </li>
      ) : null}
      {search.showLoadingMore ? (
        <li>
          <div
            role="status"
            aria-label={BACKLOG_COLUMN_SEARCH_COPY.loadingMore}
            className="flex items-center justify-center gap-2 py-2 text-[12px] text-muted-foreground"
          >
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
            <span>{BACKLOG_COLUMN_SEARCH_COPY.loadingMore}</span>
          </div>
        </li>
      ) : null}
      {search.showLoadError ? (
        <li>
          <div className="flex flex-col items-center gap-2 px-1 py-2">
            <p className="text-center text-[12px] text-destructive" role="alert">
              {BACKLOG_COLUMN_SEARCH_COPY.loadError}
            </p>
            <Button type="button" variant="outline" size="sm" onClick={search.retryLoadMore}>
              {BACKLOG_COLUMN_SEARCH_COPY.retry}
            </Button>
          </div>
        </li>
      ) : null}
      {atCapacity || search.searchActive ? null : (
        <li data-testid="lines-backlog-create-ghost-item">
          <BacklogCreatePopover variant="ghost" {...createPopover} />
        </li>
      )}
    </LineBoardColumnCardList>
  );
}

type BacklogCreatePopoverProps = ReturnType<typeof backlogCreatePopoverProps>;

async function runBacklogRefresh(run: () => Promise<RefreshBacklogResult>): Promise<void> {
  try {
    const toast = backlogRefreshToast(await run());
    if (toast.kind === "error") {
      showErrorToast(toast.message);
      return;
    }
    if (toast.kind === "success") {
      showSuccessToast(toast.message);
      return;
    }
    showInfoToast(toast.message);
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, BACKLOG_REFRESH_COPY.failed));
  }
}

function backlogCreatePopoverProps(args: {
  canAdd: boolean;
  atCapacity: boolean;
  createMenu: ReturnType<typeof useBacklogCreateMenu>;
  onCreateWorkOrder: () => void;
}) {
  return {
    canAdd: args.canAdd,
    atCapacity: args.atCapacity,
    sources: args.createMenu.sources,
    items: args.createMenu.items,
    query: args.createMenu.query,
    focusedIntakeId: args.createMenu.focusedIntakeId,
    onQueryChange: args.createMenu.setQuery,
    onFocusedIntakeChange: args.createMenu.setFocusedIntake,
    onCreateManually: args.onCreateWorkOrder,
    onImportItem: args.createMenu.importItem,
    isLoading: args.createMenu.isLoading,
    isLoadingMore: args.createMenu.isLoadingMore,
    hasMore: args.createMenu.hasMore,
    onLoadMore: args.createMenu.loadMore,
    errorMessage: args.createMenu.errorMessage,
  };
}
