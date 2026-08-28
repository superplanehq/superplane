import type { FactoriesWorkOrder } from "@/api-client";

import { BacklogCreatePopover } from "./BacklogCreatePopover";
import { BacklogSettingsDialog } from "./BacklogSettingsDialog";
import { BACKLOG_INTAKE_EMPTY_HINT, shouldShowBacklogIntakeEmptyHint } from "./backlogIntakeEmptyHint";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import { LineBoardOrderCard } from "./LineBoardOrderCard";
import { lineBoardColumnLaneClassName, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { isFirstRunOnboardingFactory } from "./lineIntakeModel";
import { BacklogOnboardingCard } from "./onboarding/first-run/BacklogOnboardingCard";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";
import { WorkOrderBoardLane, workOrderKanbanLaneScrollClassName } from "../workOrders/WorkOrderBoardChrome";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";

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
  hasIntake: boolean;
  onShowIntake: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
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
  hasIntake,
  onShowIntake,
  workOrderCardContext,
  onOpenWorkOrder,
}: BacklogColumnProps) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);
  const atCapacity = size != null && orders.length >= size;
  const canAdd = canCreateWorkOrder && !atCapacity;
  const onboarding = isFirstRunOnboardingFactory(factoryKey);
  const showIntakeHint = shouldShowBacklogIntakeEmptyHint({
    empty: orders.length === 0,
    hasIntake,
    onboarding,
  });
  const createMenu = useBacklogCreateMenu(organizationId, factoryId, onOpenWorkOrder);
  const createPopover = {
    canAdd,
    atCapacity,
    sources: createMenu.sources,
    items: createMenu.items,
    query: createMenu.query,
    focusedIntakeId: createMenu.focusedIntakeId,
    onQueryChange: createMenu.setQuery,
    onFocusedIntakeChange: createMenu.setFocusedIntake,
    onCreateManually: onCreateWorkOrder,
    onImportItem: createMenu.importItem,
    isLoading: createMenu.isLoading,
    isLoadingMore: createMenu.isLoadingMore,
    hasMore: createMenu.hasMore,
    onLoadMore: createMenu.loadMore,
    errorMessage: createMenu.errorMessage,
  };

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
        emptyContent={onboarding ? <BacklogOnboardingCard /> : undefined}
        keepChildrenWhenEmpty
        className={surfaceClassName ? undefined : "bg-muted"}
        actions={
          <div className="flex shrink-0 items-center gap-0.5">
            <BacklogCreatePopover {...createPopover} />
            <ColumnLaneMenu
              title={title}
              testId="lines-backlog-menu"
              onEdit={onOpenSettings}
              colorId={colorId}
              onColorChange={onColorChange}
            />
          </div>
        }
        testId="lines-backlog-column"
      >
        <ul className={workOrderKanbanLaneScrollClassName} data-testid="lines-backlog-column-scroll">
          {orders.map((order) => (
            <li key={order.id}>
              <LineBoardOrderCard
                order={order}
                workOrderCardContext={workOrderCardContext}
                onOpenWorkOrder={onOpenWorkOrder}
              />
            </li>
          ))}
          {atCapacity ? null : (
            <li data-testid="lines-backlog-create-ghost-item">
              <BacklogCreatePopover variant="ghost" {...createPopover} />
            </li>
          )}
          {showIntakeHint ? (
            <li>
              <p
                className="px-1 pt-1 text-[12px] leading-5 text-muted-foreground"
                data-testid="lines-backlog-intake-empty-hint"
              >
                <button
                  type="button"
                  onClick={onShowIntake}
                  aria-label="Show Intake"
                  className="underline underline-offset-2 hover:text-foreground"
                >
                  {BACKLOG_INTAKE_EMPTY_HINT.linkLabel}
                </button>
                {BACKLOG_INTAKE_EMPTY_HINT.afterLink}
              </p>
            </li>
          ) : null}
        </ul>
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
