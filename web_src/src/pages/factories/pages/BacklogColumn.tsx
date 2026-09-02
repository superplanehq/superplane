import type { FactoriesWorkOrder } from "@/api-client";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkOrderBoardLane, workOrderKanbanLaneScrollClassName } from "../workOrders/WorkOrderBoardChrome";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { BacklogCreatePopover } from "./BacklogCreatePopover";
import { BacklogIntakeSources } from "./BacklogIntakeSources";
import { BacklogSettingsDialog } from "./BacklogSettingsDialog";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import { CreateWithAgentDialog } from "./CreateWithAgentDialog";
import { LineBoardOrderCard } from "./LineBoardOrderCard";
import { lineBoardColumnLaneClassName, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { isFirstRunOnboardingFactory, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { BacklogOnboardingCard } from "./onboarding/first-run/BacklogOnboardingCard";
import { workspacePlanningRepository } from "./planningSessionView";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";
import { useCreateWithAgentSession } from "./useCreateWithAgentSession";

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
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrder) => void;
  /** Tasks the Backlog automation analyzes right now. */
  analyzingOrderIds?: ReadonlySet<string>;
  /** Intakes that open tasks in this backlog, listed at its head. */
  intakePanel?: BacklogIntakePanel;
  /** Configure link for the factory Backlog automation, when one exists. */
  automationHref?: string | null;
  /** Opens the Add intake picker from the overflow menu. Hidden when unset. */
  onAddIntake?: () => void;
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
  workOrderCardContext,
  onOpenWorkOrder,
  analyzingOrderIds,
  intakePanel,
  automationHref,
  onAddIntake,
}: BacklogColumnProps) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);
  const atCapacity = size != null && orders.length >= size;
  const canAdd = canCreateWorkOrder && !atCapacity;
  const { factory } = useFactoriesLayout();
  const createMenu = useBacklogCreateMenu(organizationId, factoryId, onOpenWorkOrder);
  const agentSession = useCreateWithAgentSession(workspacePlanningRepository(factory), organizationId, factoryId);
  const createPopover = backlogCreatePopoverProps({
    canAdd,
    atCapacity,
    createMenu,
    onCreateWorkOrder,
    onCreateWithAgent: agentSession.start,
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
          <div className="flex shrink-0 items-center gap-0.5">
            <BacklogCreatePopover {...createPopover} />
            <ColumnLaneMenu
              title={title}
              testId="lines-backlog-menu"
              automationHref={automationHref}
              onEdit={onOpenSettings}
              onAddIntake={onAddIntake}
              colorId={colorId}
              onColorChange={onColorChange}
            />
          </div>
        }
        banner={
          intakePanel ? (
            <BacklogIntakeSources
              intakes={intakePanel.sources}
              showAddIntake={intakePanel.showAddIntake}
              onOpenSettings={intakePanel.onOpenSettings}
              onAddIntake={intakePanel.onAddIntake}
            />
          ) : null
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
      </WorkOrderBoardLane>
      <BacklogSettingsDialog
        open={settingsOpen}
        name={title}
        size={size}
        onSave={onSaveSettings}
        onClose={onCloseSettings}
      />
      <CreateWithAgentDialog
        open={agentSession.open}
        workspaceName={factoryKey}
        organizationId={organizationId}
        view={agentSession.view}
        onComposerChange={agentSession.onComposerChange}
        onSend={agentSession.onSend}
        onSubmitSurvey={agentSession.onSubmitSurvey}
        onDraftTitleChange={agentSession.onDraftTitleChange}
        onDraftDescriptionChange={agentSession.onDraftDescriptionChange}
        onCreateDraft={agentSession.onCreateDraft}
        onSkipDraft={agentSession.onSkipDraft}
        onOpenCreated={(order) => {
          agentSession.close();
          onOpenWorkOrder(order.id);
        }}
        onRequestClose={agentSession.onRequestClose}
        onCancelEnd={agentSession.onCancelEnd}
        onConfirmEnd={agentSession.onConfirmEnd}
      />
    </>
  );
}

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
