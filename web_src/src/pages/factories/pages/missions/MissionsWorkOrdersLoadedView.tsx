import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import { createWorkOrderPath } from "../../lib/factoryPagePaths";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import {
  applyWorkOrderFilters,
  applyWorkOrderOrdering,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  buildWorkOrderListEntries,
} from "../../lib/workOrderListModel";
import { factoryContentBodyClassName, factoryContentHeaderClassName } from "../factoryPageLayoutStyles";
import {
  WorkOrdersFilteredEmptyState,
  WorkOrdersScopedEmptyState,
  WorkOrdersTrueEmptyState,
} from "../../workOrders/WorkOrdersEmptyStates";
import { WorkOrdersBoardView } from "../../workOrders/WorkOrdersBoardView";
import { WorkOrdersHeader } from "../../workOrders/header/WorkOrdersHeader";
import { WorkOrdersListView } from "../../workOrders/WorkOrdersListView";
import { WorkOrdersTableView } from "../../workOrders/WorkOrdersTableView";
import { useMissionAssignment } from "./useMissionAssignment";
import { buildMissionRailItems } from "./missionListModel";
import { MissionViews } from "./MissionViews";

export interface MissionsWorkOrdersLoadedViewProps {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factoryLines: FactoriesFactoryLine[];
  workOrders: FactoriesWorkOrder[];
  state: WorkOrderListState;
  currentUserId?: string;
  canCreate: boolean;
  canDispatch: boolean;
  canAssign: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

export function MissionsWorkOrdersLoadedView(props: MissionsWorkOrdersLoadedViewProps) {
  const { workOrders, factory, state, currentUserId } = props;
  const { missions, missionByWorkOrderId, closedByMissionId } = useMissionAssignment();
  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [workOrders, factory]);
  const scoped = useMemo(
    () => applyWorkOrderScope(entries, state.scope, currentUserId),
    [entries, state.scope, currentUserId],
  );
  const filtered = useMemo(() => applyWorkOrderFilters(scoped, state.filters), [scoped, state.filters]);
  const searched = useMemo(() => applyWorkOrderSearch(filtered, state.search), [filtered, state.search]);
  const ordered = useMemo(() => applyWorkOrderOrdering(searched, state.ordering), [searched, state.ordering]);
  const railItems = useMemo(
    () => buildMissionRailItems(missions, entries, missionByWorkOrderId, closedByMissionId),
    [closedByMissionId, entries, missionByWorkOrderId, missions],
  );

  const totalCount = entries.length;
  const createHref = createWorkOrderPath(props.organizationId, props.factoryKey);

  const body = () => {
    if (totalCount === 0) {
      return (
        <WorkOrdersTrueEmptyState
          createHref={createHref}
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
      <header className={factoryContentHeaderClassName}>
        <WorkOrdersHeader
          state={state}
          entries={entries}
          factoryLines={props.factoryLines}
          createHref={createHref}
          canCreate={props.canCreate}
          permissionsLoading={props.permissionsLoading}
        />
      </header>

      <div className={cn(factoryContentBodyClassName, "flex flex-col gap-4")}>
        {totalCount > 0 ? (
          <MissionViews
            items={railItems}
            layout={state.layout}
            scope={state.scope}
            currentUserId={currentUserId}
            organizationId={props.organizationId}
            factoryKey={props.factoryKey}
          />
        ) : null}
        {body()}
      </div>
    </>
  );
}
