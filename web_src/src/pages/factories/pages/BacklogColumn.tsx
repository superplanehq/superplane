import type { FactoriesWorkOrder } from "@/api-client";

import { WorkOrderBoardLane, workOrderKanbanLaneScrollClassName } from "../workOrders/WorkOrderBoardChrome";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { BacklogCreatePopover } from "./BacklogCreatePopover";
import { BacklogIntakeSources } from "./BacklogIntakeSources";
import { BacklogSettingsDialog } from "./BacklogSettingsDialog";
import { ColumnAutomationsHeaderSlot } from "./ColumnAutomationsIndicator";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import type { ColumnAutomation } from "../lib/columnAutomations";
import type { ColumnAutomationRowAction } from "./ColumnAutomationsPopup";
import { LineBoardOrderCard } from "./LineBoardOrderCard";
import { lineBoardColumnLaneClassName, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { isFirstRunOnboardingFactory, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { BacklogOnboardingCard } from "./onboarding/first-run/BacklogOnboardingCard";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";

export type BacklogColumnProps = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  orders: FactoriesWorkOrder[];
  title: string;
  size: number | null;
  settingsOpen: boolean;
  onOpenSettings: () => void;
  onCloseSettings: () => void;
  onSaveSettings: (settings: { name: string; size: number | null }) => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onRename: (title: string) => void;
  onCreateWorkOrder: () => void;
  onCreateWithAgent: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  /** Tasks the Backlog automation analyzes right now. */
  analyzingOrderIds?: ReadonlySet<string>;
  /** Intakes that open tasks in this backlog, listed at its head. */
  intakePanel?: BacklogIntakePanel;
  /** Opens the Add intake picker from the overflow menu. Hidden when unset. */
  onAddIntake?: () => void;
  /** Column automations for the header icons. Hidden when unset. */
  automations?: ColumnAutomation[];
  onAddAutomation?: () => void;
  onAutomationRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
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
  orders,
  title,
  size,
  settingsOpen,
  onOpenSettings,
  onCloseSettings,
  onSaveSettings,
  colorId,
  onColorChange,
  canCreateWorkOrder,
  canRename,
  onRename,
  onCreateWorkOrder,
  onCreateWithAgent,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  intakePanel,
  onAddIntake,
  automations,
  onAddAutomation,
  onAutomationRowAction,
}: BacklogColumnProps) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);
  const atCapacity = size != null && orders.length >= size;
  const canAdd = canCreateWorkOrder && !atCapacity;
  const createMenu = useBacklogCreateMenu(organizationId, factoryId, onOpenWorkOrder);
  const createPopover = backlogCreatePopoverProps({
    canAdd,
    atCapacity,
    createMenu,
    onCreateWorkOrder,
    onCreateWithAgent,
  });

  return (
    <>
      <WorkOrderBoardLane
        title={title}
        label={title}
        canRename={canRename}
        onRename={onRename}
        titleTestId="lines-column-title-backlog"
        count={orders.length}
        tone="neutral"
        surfaceClassName={surfaceClassName}
        emptyDescription="No tasks in the backlog."
        emptyContent={isFirstRunOnboardingFactory(factoryKey) ? <BacklogOnboardingCard /> : undefined}
        keepChildrenWhenEmpty
        className={surfaceClassName ? undefined : "bg-muted"}
        actions={
          <BacklogColumnHeaderActions
            title={title}
            createPopover={createPopover}
            automations={automations}
            onAddAutomation={onAddAutomation}
            onAutomationRowAction={onAutomationRowAction}
            onOpenSettings={onOpenSettings}
            onAddIntake={onAddIntake}
            colorId={colorId}
            onColorChange={onColorChange}
          />
        }
        banner={intakePanel ? <BacklogColumnIntakeBanner panel={intakePanel} /> : null}
        testId="lines-backlog-column"
      >
        <BacklogColumnOrderList
          orders={orders}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
          analyzingOrderIds={analyzingOrderIds}
          atCapacity={atCapacity}
          createPopover={createPopover}
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

function BacklogColumnHeaderActions({
  title,
  createPopover,
  automations,
  onAddAutomation,
  onAutomationRowAction,
  onOpenSettings,
  onAddIntake,
  colorId,
  onColorChange,
}: Pick<
  BacklogColumnProps,
  | "title"
  | "automations"
  | "onAddAutomation"
  | "onAutomationRowAction"
  | "onOpenSettings"
  | "onAddIntake"
  | "colorId"
  | "onColorChange"
> & {
  createPopover: BacklogCreatePopoverProps;
}) {
  return (
    <div className="flex shrink-0 items-center gap-0.5">
      <BacklogCreatePopover {...createPopover} />
      <ColumnAutomationsHeaderSlot
        title={title}
        automations={automations}
        onRowAction={onAutomationRowAction}
        testId="lines-backlog-automations"
      />
      <ColumnLaneMenu
        title={title}
        testId="lines-backlog-menu"
        onEdit={onOpenSettings}
        onAddIntake={onAddIntake}
        onAddAutomation={onAddAutomation}
        colorId={colorId}
        onColorChange={onColorChange}
      />
    </div>
  );
}

function BacklogColumnIntakeBanner({ panel }: { panel: BacklogIntakePanel }) {
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
  atCapacity,
  createPopover,
}: Pick<BacklogColumnProps, "orders" | "workOrderCardContext" | "onOpenWorkOrder" | "analyzingOrderIds"> & {
  atCapacity: boolean;
  createPopover: BacklogCreatePopoverProps;
}) {
  return (
    <ul className={workOrderKanbanLaneScrollClassName} data-testid="lines-backlog-column-scroll">
      {orders.map((order) => (
        <li key={order.id}>
          <LineBoardOrderCard
            order={order}
            workOrderCardContext={workOrderCardContext}
            onOpenWorkOrder={onOpenWorkOrder}
            isAnalyzing={Boolean(order.id && analyzingOrderIds?.has(order.id))}
          />
        </li>
      ))}
      {atCapacity ? null : (
        <li data-testid="lines-backlog-create-ghost-item">
          <BacklogCreatePopover variant="ghost" {...createPopover} />
        </li>
      )}
    </ul>
  );
}

type BacklogCreatePopoverProps = ReturnType<typeof backlogCreatePopoverProps>;

function backlogCreatePopoverProps(args: {
  canAdd: boolean;
  atCapacity: boolean;
  createMenu: ReturnType<typeof useBacklogCreateMenu>;
  onCreateWorkOrder: () => void;
  onCreateWithAgent: () => void;
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
    onCreateWithAgent: args.onCreateWithAgent,
    onImportItem: args.createMenu.importItem,
    isLoading: args.createMenu.isLoading,
    isLoadingMore: args.createMenu.isLoadingMore,
    hasMore: args.createMenu.hasMore,
    onLoadMore: args.createMenu.loadMore,
    errorMessage: args.createMenu.errorMessage,
  };
}
