import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
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
import { factoryContentBodyClassName } from "../pages/factoryPageLayoutStyles";
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
  state: WorkOrderListState;
  currentUserId?: string;
  canCreate: boolean;
  onCreateWorkOrder: () => void;
  canDispatch: boolean;
  canAssign: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Data-agnostic Work Orders view. Receives raw work orders + the shared
 * `WorkOrderListState` and renders the toolbar + selected layout. Kept
 * separate from `WorkOrdersPage` so stories can drive it with fixtures
 * and the shell page only handles fetching + mutations.
 */
export function WorkOrdersLoadedView(props: WorkOrdersLoadedViewProps) {
  const { workOrders, factory, state, currentUserId } = props;
  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [workOrders, factory]);
  const scoped = useMemo(
    () => applyWorkOrderScope(entries, state.scope, currentUserId),
    [entries, state.scope, currentUserId],
  );
  const filtered = useMemo(() => applyWorkOrderFilters(scoped, state.filters), [scoped, state.filters]);
  const searched = useMemo(() => applyWorkOrderSearch(filtered, state.search), [filtered, state.search]);
  const ordered = useMemo(() => applyWorkOrderOrdering(searched, state.ordering), [searched, state.ordering]);

  const totalCount = entries.length;

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
      isDispatching: props.isDispatching,
      isAssigneesSaving: props.isAssigneesSaving,
      onDispatch: props.onDispatch,
      onAssigneesSave: props.onAssigneesSave,
    };

    if (state.layout === "board") {
      return <WorkOrdersBoardView {...sharedProps} />;
    }
    if (state.layout === "list") {
      return <WorkOrdersListView {...sharedProps} />;
    }
    return <WorkOrdersTableView {...sharedProps} />;
  };

  return (
    <>
      <WorkOrdersHeader
        state={state}
        entries={entries}
        factoryLines={props.factoryLines}
        onCreateWorkOrder={props.onCreateWorkOrder}
        canCreate={props.canCreate}
        permissionsLoading={props.permissionsLoading}
      />

      <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>{body()}</div>
    </>
  );
}
