import type { FactoriesWorkOrder } from "@/api-client";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { type RefreshBacklogResult, useFactoryIntakes, useRefreshBacklog } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

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
import { LineBoardOrderCard } from "./LineBoardOrderCard";
import type { LineBoardColumnColorView } from "../lib/lineBoardColumnColorViewPreference";
import { lineBoardColumnLaneProps, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { isFirstRunOnboardingFactory, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { BacklogOnboardingCard } from "./onboarding/first-run/BacklogOnboardingCard";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";
import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";
import { BACKLOG_REFRESH_COPY, backlogRefreshToast, canRefreshBacklog } from "./backlogRefresh";

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
  /** Intakes that open tasks in this backlog, listed at its head. */
  intakePanel?: BacklogIntakePanel;
  /** Opens the Add intake picker from the overflow menu. Hidden when unset. */
  onAddIntake?: () => void;
  /** Opens guided Sentry intake setup. Hidden when unset. */
  onSetupSentry?: () => void;
  /** Opens guided Jira intake setup. Hidden when unset. */
  onSetupJira?: () => void;
  /** Column automations for the header icons. Hidden when unset. */
  automations?: ColumnAutomation[];
  /** Rows the automation subheader reserves. Shared across the board. Hidden when unset. */
  automationRowCount?: number;
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
  colorView,
  onColorChange,
  canCreateWorkOrder,
  canRename,
  onRename,
  onCreateWorkOrder,
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  intakePanel,
  onAddIntake,
  onSetupSentry,
  onSetupJira,
  automations,
  automationRowCount,
  onAutomationRowAction,
}: BacklogColumnProps) {
  const lane = lineBoardColumnLaneProps(colorId, colorView, { mutedFallback: true });
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
        surfaceClassName={lane.surfaceClassName}
        emptyDescription="No tasks in the backlog."
        emptyContent={isFirstRunOnboardingFactory(factoryKey) ? <BacklogOnboardingCard /> : undefined}
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
            onRefreshBacklog={
              canRefreshBacklog(intakesQuery.data, canUpdateWorkOrders)
                ? () => {
                    void runBacklogRefresh(refreshBacklog.mutateAsync);
                  }
                : undefined
            }
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
        banner={<BacklogColumnBanner panel={intakePanel} onSetupSentry={onSetupSentry} onSetupJira={onSetupJira} />}
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

function BacklogColumnBanner({
  panel,
  onSetupSentry,
  onSetupJira,
}: {
  panel?: BacklogIntakePanel;
  onSetupSentry?: () => void;
  onSetupJira?: () => void;
}) {
  if (!panel && !onSetupSentry && !onSetupJira) {
    return null;
  }

  return (
    <>
      {panel ? (
        <BacklogIntakeSources
          intakes={panel.sources}
          showAddIntake={panel.showAddIntake}
          onOpenSettings={panel.onOpenSettings}
          onAddIntake={panel.onAddIntake}
        />
      ) : null}
      {onSetupSentry ? <BacklogSetupSentryButton onClick={onSetupSentry} /> : null}
      {onSetupJira ? <BacklogSetupJiraButton onClick={onSetupJira} /> : null}
    </>
  );
}

function BacklogSetupSentryButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={onClick}
      data-testid="lines-backlog-setup-sentry"
      className="mb-2 h-8 w-full justify-start gap-2 px-2 text-[12px] font-medium tracking-[-0.01em]"
    >
      <img
        src={sentryIcon}
        alt=""
        className={cn("size-3.5 shrink-0 object-contain", logoDarkInvertClass(sentryIcon))}
      />
      {SENTRY_INTAKE_SETUP_COPY.setupButton}
    </Button>
  );
}

function BacklogSetupJiraButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={onClick}
      data-testid="lines-backlog-setup-jira"
      className="mb-2 h-8 w-full justify-start gap-2 px-2 text-[12px] font-medium tracking-[-0.01em]"
    >
      <img src={jiraIcon} alt="" className="size-3.5 shrink-0 object-contain" />
      {JIRA_INTAKE_SETUP_COPY.setupButton}
    </Button>
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
