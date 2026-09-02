import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { useMemo, type ReactNode } from "react";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import {
  applyWorkOrderFilters,
  applyWorkOrderOrdering,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  buildWorkOrderListEntries,
} from "../../lib/workOrderListModel";
import { factoryKanbanPageClassName, factoryWorkOrdersBodyClassName } from "../factoryPageLayoutStyles";
import {
  WorkOrdersFilteredEmptyState,
  WorkOrdersScopedEmptyState,
  WorkOrdersTrueEmptyState,
} from "../../workOrders/WorkOrdersEmptyStates";
import { WorkOrdersBoardView } from "../../workOrders/WorkOrdersBoardView";
import { WorkOrdersHeader } from "../../workOrders/header/WorkOrdersHeader";
import { WorkOrdersListView } from "../../workOrders/WorkOrdersListView";
import { WorkOrdersTableView } from "../../workOrders/WorkOrdersTableView";

export interface MissionsWorkOrdersLoadedViewProps {
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
  /** Tasks with a dispatch in flight. Only their controls show a busy state. */
  dispatchingOrderIds: ReadonlySet<string>;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
  hostedCreditEmptyBanner?: ReactNode;
}

export function MissionsWorkOrdersLoadedView(props: MissionsWorkOrdersLoadedViewProps) {
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
      return <WorkOrdersBoardView {...sharedProps} />;
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
          hostedCreditEmptyBanner={props.hostedCreditEmptyBanner}
        />
      </div>

      <div className={cn(factoryWorkOrdersBodyClassName, "flex flex-col gap-4")}>{body()}</div>
    </div>
  );
}
