import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
} from "@/api-client";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import {
  applyWorkOrderFilters,
  applyWorkOrderOrdering,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  buildWorkOrderListEntries,
} from "../lib/workOrderListModel";
import type { WorkOrderListState } from "../lib/useWorkOrderListState";
import { factoryKanbanPageClassName, factoryWorkOrdersBodyClassName } from "../pages/factoryPageLayoutStyles";
import { usePRFeedbackWorkOrderAttention } from "../pages/useWorkOrderPRFeedbackRunHref";
import { WorkOrdersBoardView } from "./WorkOrdersBoardView";
import {
  WorkOrdersFilteredEmptyState,
  WorkOrdersScopedEmptyState,
  WorkOrdersTrueEmptyState,
} from "./WorkOrdersEmptyStates";
import { WorkOrdersHeader } from "./header/WorkOrdersHeader";
import { WorkOrdersListView } from "./WorkOrdersListView";
import { WorkOrdersTableView } from "./WorkOrdersTableView";

interface WorkOrdersLoadedViewProps {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factoryLines: FactoriesFactoryLine[];
  workOrders: FactoriesWorkOrder[];
  pullRequests?: FactoriesFactoryPullRequest[];
  state: WorkOrderListState;
  currentUserId?: string;
  canCreate: boolean;
  onCreateWorkOrder: () => void;
  canDispatch: boolean;
  canAssign: boolean;
  permissionsLoading: boolean;
  /** Tasks with a dispatch in flight. Only their controls show a busy state. */
  dispatchingOrderIds: ReadonlySet<string>;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Data-agnostic Tasks view. Receives raw tasks + the shared
 * `WorkOrderListState` and renders the toolbar + selected layout. Kept
 * separate from `WorkOrdersPage` so stories can drive it with fixtures
 * and the shell page only handles fetching + mutations.
 */
export function WorkOrdersLoadedView(props: WorkOrdersLoadedViewProps) {
  const { workOrders, factory, state, currentUserId, pullRequests = [] } = props;
  const { addressingFeedbackOrderIds, addressingFeedbackLabels, waitingOnChecksOrderIds, checksPassedOrderIds } =
    usePRFeedbackWorkOrderAttention(pullRequests);
  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [workOrders, factory]);
  const scoped = useMemo(
    () => applyWorkOrderScope(entries, state.scope, currentUserId),
    [entries, state.scope, currentUserId],
  );
  const filtered = useMemo(() => applyWorkOrderFilters(scoped, state.filters), [scoped, state.filters]);
  const searched = useMemo(() => applyWorkOrderSearch(filtered, state.search), [filtered, state.search]);
  const ordered = useMemo(() => applyWorkOrderOrdering(searched, state.ordering), [searched, state.ordering]);

  const totalCount = entries.length;
  const showKanbanBoard = state.layout === "board" && totalCount > 0 && ordered.length > 0;

  const body = () => {
    if (totalCount === 0) {
      return (
        <WorkOrdersTrueEmptyState
          onCreateWorkOrder={props.onCreateWorkOrder}
          canCreate={props.canCreate}
          permissionsLoading={props.permissionsLoading}
        />
      );
    }
    if (ordered.length === 0) {
      if (state.scope !== "all" && state.filterCount === 0 && state.search.trim().length === 0) {
        return (
          <WorkOrdersScopedEmptyState
            scopeLabel={state.scope === "my" ? "your work" : "active work"}
            onResetScope={() => state.setScope("all")}
          />
        );
      }
      return <WorkOrdersFilteredEmptyState onClearFilters={state.resetView} />;
    }

    const sharedProps = {
      entries: ordered,
      organizationId: props.organizationId,
      factoryKey: props.factoryKey,
      factoryLines: props.factoryLines,
      canDispatch: props.canDispatch,
      canAssign: props.canAssign,
      dispatchingOrderIds: props.dispatchingOrderIds,
      isAssigneesSaving: props.isAssigneesSaving,
      onDispatch: props.onDispatch,
      onAssigneesSave: props.onAssigneesSave,
    };

    if (state.layout === "board") {
      return (
        <WorkOrdersBoardView
          {...sharedProps}
          factoryId={factory.id}
          addressingFeedbackOrderIds={addressingFeedbackOrderIds}
          addressingFeedbackLabels={addressingFeedbackLabels}
          waitingOnChecksOrderIds={waitingOnChecksOrderIds}
          checksPassedOrderIds={checksPassedOrderIds}
        />
      );
    }
    if (state.layout === "list") {
      return <WorkOrdersListView {...sharedProps} />;
    }
    return <WorkOrdersTableView {...sharedProps} />;
  };

  return (
    <div className={showKanbanBoard ? factoryKanbanPageClassName : undefined}>
      <div className="shrink-0">
        <WorkOrdersHeader
          state={state}
          entries={entries}
          factoryLines={props.factoryLines}
          onCreateWorkOrder={props.onCreateWorkOrder}
          canCreate={props.canCreate}
          permissionsLoading={props.permissionsLoading}
        />
      </div>

      <div className={cn(factoryWorkOrdersBodyClassName, "flex flex-col gap-4")}>{body()}</div>
    </div>
  );
}
