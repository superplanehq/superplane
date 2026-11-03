import { PullRequestImportPicker } from "../pages/PullRequestImportPicker";
import { SubscriptionIntakeOwnerRecovery } from "../pages/SubscriptionIntakeOwnerRecovery";
import type { FactoriesFactoryIntake, FactoriesFactoryLine, FactoriesWorkOrderSummary } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import {
  useFactoryAutomations,
  useFactoryBoardWorkOrders,
  type FactoryBoardColumnPage,
  type FactoryBoardWorkOrdersOptions,
} from "@/hooks/useFactoryData";
import { useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { FEATURE_FACTORY_PULL_REQUEST_MERGE } from "@/lib/experimentalFeatures";
import { getOrgUserDisplayFromUser } from "@/lib/orgUserDisplay";
import { useMemo } from "react";
import { Navigate, useLocation, useNavigate, useParams } from "react-router";

import { backlogAnalysisCreditLabels } from "../lib/backlogAnalysis";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { useLineBoardColumnColorViewPreference } from "../lib/lineBoardColumnColorViewPreference";
import {
  factoryHomePath,
  factoryLineDetailPath,
  intakeIdFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  isPlanningSearchOpen,
  columnAutomationViewCanvasIdFromSearch,
  firstFactoryLineId,
  workOrderDetailPath,
} from "../lib/factoryPagePaths";
import { boardDoneResultsForStatuses, uniqueWorkOrdersById } from "../lib/workOrderListPagination";
import { useWorkOrderListState, type WorkOrderListState } from "../lib/useWorkOrderListState";
import { useWorkOrdersHeaderShortcuts } from "../lib/useWorkOrdersHeaderShortcuts";
import {
  buildAssigneeFilterOptions,
  buildSourceFilterOptions,
  type WorkOrderFilterOption,
} from "../lib/workOrderFilterOptions";
import {
  applyWorkOrderFilters,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  boardWorkOrderScopeQuery,
  buildWorkOrderListEntries,
  type WorkOrderScope,
} from "../lib/workOrderListModel";
import { canonicalWorkOrderNumber } from "../lib/workOrderNumberResolution";
import { pullRequestsFromWorkOrders } from "../lib/workOrderPullRequest";
import { LineBoardWorkOrderCard } from "../pages/LineBoardOrderCard";
import { lineBoardColumnLaneProps, normalizeColumnColors } from "../pages/lineBoardColumnColors";
import { intakeSourcesFromFactoryIntakes } from "../pages/lineIntakeModel";
import { usePRFeedbackWorkOrderAttention } from "../pages/useWorkOrderPRFeedbackRunHref";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import type { ColumnPaging } from "./MobileBoardColumn";
import { MobileColumnCarousel } from "./MobileColumnCarousel";
import { MobileBoardHeader } from "./MobileBoardHeader";
import { MobileBoardSettings } from "./MobileBoardSettings";
import { MobileCreateTaskButton } from "./MobileCreateTaskButton";
import { buildMobileBoardColumns, type MobileBoardColumn } from "./mobileBoardColumns";

function phoneBoardScope(scope: WorkOrderScope): WorkOrderScope {
  if (scope === "my" || scope === "unassigned") {
    return scope;
  }
  return "all";
}

function boardQueryOptions(
  state: WorkOrderListState,
  lineId: string,
  currentUserId?: string,
): FactoryBoardWorkOrdersOptions {
  const scope = phoneBoardScope(state.scope);
  return {
    ...boardWorkOrderScopeQuery(scope, state.filters.assigneeIds, currentUserId),
    done: { lineId, results: boardDoneResultsForStatuses(state.filters.statuses) },
  };
}

/** Phone line board: one column in view, swipe sideways to change column. */
export function MobileBoardPage() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const { lineId: routeLineId } = useParams<{ lineId?: string }>();
  const navigate = useNavigate();
  const { canAct, currentUserId } = usePermissions();
  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const selectedLine = lines.find((line) => line.id === routeLineId) ?? null;

  usePageTitle([factory?.name?.trim() || "Workspace"]);

  if (!selectedLine?.id) {
    return <Navigate to={factoryHomePath(organizationId, routeSegment, firstFactoryLineId(factory))} replace />;
  }

  return (
    <MobileLineBoard
      organizationId={organizationId}
      factoryId={factoryId}
      routeSegment={routeSegment}
      line={selectedLine}
      lineId={selectedLine.id}
      canUpdateWorkOrders={canAct("work_orders", "update")}
      currentUserId={currentUserId}
      onOpenWorkOrder={(order, columnKey) => {
        const number = canonicalWorkOrderNumber(order);
        if (number) {
          navigate(workOrderDetailPath(organizationId, routeSegment, number), {
            state: { peekOrder: order, lineId: selectedLine.id, column: columnKey },
          });
        }
      }}
    />
  );
}

type MobileBoardModel = {
  listState: WorkOrderListState;
  showPullRequestMerge: boolean;
  columns: MobileBoardColumn[];
  cardsPending: boolean;
  paging: Record<MobileBoardColumn["paging"], ColumnPaging>;
  sourceOptions: WorkOrderFilterOption[];
  assigneeOptions: WorkOrderFilterOption[];
  factoryIntakes: FactoriesFactoryIntake[];
  apps: Array<{ id?: string; name?: string }>;
  workOrderCardContext: WorkOrderCardContext;
  analyzingOrderIds: Set<string>;
  creditFailureLabels: ReadonlyMap<string, string>;
};

function columnPaging(page: FactoryBoardColumnPage, isPlaceholderData: boolean): ColumnPaging {
  return {
    hasMore: !isPlaceholderData && page.hasNextPage,
    isLoading: page.isFetchingNextPage,
    hasPageError: page.isFetchNextPageError,
    onLoadMore: page.fetchNextPage,
  };
}

/** Loads the board data and applies scope, filters, and search to the visible columns. */
function useMobileBoardModel({
  organizationId,
  factoryId,
  routeSegment,
  line,
  lineId,
  canUpdateWorkOrders,
  currentUserId,
}: {
  organizationId: string;
  factoryId: string;
  routeSegment: string;
  line: FactoriesFactoryLine;
  lineId: string;
  canUpdateWorkOrders: boolean;
  currentUserId?: string;
}): MobileBoardModel {
  const { factory } = useFactoriesLayout();
  const listState = useWorkOrderListState(factoryId);
  const { has: hasExperimentalFeature } = useExperimentalFeature(organizationId);
  const showPullRequestMerge = hasExperimentalFeature(FEATURE_FACTORY_PULL_REQUEST_MERGE);

  const { workOrders, isPlaceholderData, backlog, open, done } = useFactoryBoardWorkOrders(
    organizationId,
    factoryId,
    boardQueryOptions(listState, lineId, currentUserId),
  );
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const { data: factoryIntakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const { data: prFeedbackHandlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const { data: orgUsers = [] } = useOrganizationUsers(organizationId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const backlogAnalysis = useFactoryBacklogAnalysis(organizationId, factoryId);
  const pullRequests = useMemo(() => pullRequestsFromWorkOrders(workOrders), [workOrders]);
  const attention = usePRFeedbackWorkOrderAttention(pullRequests, prFeedbackHandlers);

  const entries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [factory, workOrders]);
  const visibleWorkOrders = useMemo(() => {
    const visibleIds = new Set(
      applyWorkOrderSearch(
        applyWorkOrderFilters(
          applyWorkOrderScope(entries, phoneBoardScope(listState.scope), currentUserId),
          { ...listState.filters, lineIds: [] },
          { showPullRequestMerge },
        ),
        listState.search,
      ).map((entry) => entry.id),
    );
    return uniqueWorkOrdersById(workOrders.filter((order) => order.id && visibleIds.has(order.id)));
  }, [currentUserId, entries, listState.filters, listState.scope, listState.search, showPullRequestMerge, workOrders]);

  const hasClientFilter =
    Boolean(listState.search.trim()) || listState.filterCount - listState.filters.lineIds.length > 0;

  const totalCounts = useMemo(() => {
    if (hasClientFilter) {
      return undefined;
    }
    return {
      backlog: backlog.totalCount,
      done: done.totalCount,
    };
  }, [backlog.totalCount, done.totalCount, hasClientFilter]);

  const columns = useMemo(
    () => buildMobileBoardColumns(line, visibleWorkOrders, apps, totalCounts),
    [apps, line, totalCounts, visibleWorkOrders],
  );
  const sourceOptions = useMemo(() => buildSourceFilterOptions(factoryIntakes, entries), [entries, factoryIntakes]);
  const assigneeOptions = useMemo(
    () =>
      buildAssigneeFilterOptions(
        entries,
        orgUsers.flatMap((user) => {
          const display = getOrgUserDisplayFromUser(user);
          return display ? [{ id: display.id, name: display.name }] : [];
        }),
      ),
    [entries, orgUsers],
  );
  const creditFailureLabels = useMemo(
    () => backlogAnalysisCreditLabels(backlogAnalysis.runsByWorkOrder),
    [backlogAnalysis.runsByWorkOrder],
  );

  return {
    listState,
    showPullRequestMerge,
    columns,
    cardsPending: Boolean(isPlaceholderData),
    paging: {
      backlog: columnPaging(backlog, Boolean(isPlaceholderData)),
      open: columnPaging(open, Boolean(isPlaceholderData)),
      done: columnPaging(done, Boolean(isPlaceholderData)),
    },
    sourceOptions,
    assigneeOptions,
    factoryIntakes,
    apps,
    workOrderCardContext: {
      organizationId,
      factoryId,
      factoryKey: routeSegment,
      factoryLines: factory?.lines ?? [],
      preferredLineName: line.name,
      canDispatch: canUpdateWorkOrders,
      canAssign: canUpdateWorkOrders,
      pullRequests,
      ...attention,
      ...cardActions,
    },
    analyzingOrderIds: backlogAnalysis.analyzingOrderIds,
    creditFailureLabels,
  };
}

function MobileLineBoard(props: {
  organizationId: string;
  factoryId: string;
  routeSegment: string;
  line: FactoriesFactoryLine;
  lineId: string;
  canUpdateWorkOrders: boolean;
  currentUserId?: string;
  onOpenWorkOrder: (order: FactoriesWorkOrderSummary, columnKey?: string) => void;
}) {
  const { organizationId, factoryId, routeSegment, line, lineId, onOpenWorkOrder } = props;
  const navigate = useNavigate();
  const { factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const model = useMobileBoardModel(props);
  const searchRef = useWorkOrdersHeaderShortcuts(model.listState);
  const { view: colorView } = useLineBoardColumnColorViewPreference();
  const columnColors = useMemo(() => normalizeColumnColors(line.columnColors), [line.columnColors]);
  const configuredIntakes = useMemo(
    () => intakeSourcesFromFactoryIntakes(model.factoryIntakes),
    [model.factoryIntakes],
  );
  const canConfigureFactory = canAct("factories", "update");
  const canCreateWorkOrder = canAct("work_orders", "create");
  const lines = factory?.lines ?? [];
  const { search } = useLocation();
  const settingsOpen =
    (isIntakeSearchOpen(search) &&
      configuredIntakes.some((intake) => intake.intakeId === intakeIdFromSearch(search))) ||
    isPRFeedbackSearchOpen(search) ||
    isPlanningSearchOpen(search) ||
    Boolean(columnAutomationViewCanvasIdFromSearch(search));

  return (
    <div className="relative flex h-full min-h-0 flex-col" data-testid="mobile-board-page">
      <MobileBoardSettings
        organizationId={organizationId}
        factoryId={factoryId}
        routeSegment={routeSegment}
        lineId={lineId}
        intakes={configuredIntakes}
        canUpdate={canConfigureFactory}
        onboarding={factory?.onboarding}
        apps={model.apps}
      />
      {!settingsOpen ? (
        <>
          <SubscriptionIntakeOwnerRecovery />
          <MobileBoardHeader
            state={model.listState}
            searchRef={searchRef}
            sourceOptions={model.sourceOptions}
            assigneeOptions={model.assigneeOptions}
            showPullRequestMerge={model.showPullRequestMerge}
            lines={lines}
            lineId={lineId}
            onSelectLine={(nextLineId) => {
              if (nextLineId !== lineId) {
                navigate(factoryLineDetailPath(organizationId, routeSegment, nextLineId));
              }
            }}
          />
          <MobileColumnCarousel
            key={lineId}
            columns={model.columns}
            cardsPending={model.cardsPending}
            laneClassName={(column) =>
              lineBoardColumnLaneProps(columnColors[column.key] ?? null, colorView, { mutedFallback: true })
            }
            paging={model.paging}
            renderColumnAction={(column) =>
              column.key === "verify" ? (
                <PullRequestImportPicker
                  mobile
                  organizationId={organizationId}
                  factoryId={factoryId}
                  lineId={lineId}
                  onOpenTask={(_, order) => {
                    if (order) onOpenWorkOrder(order, "verify");
                  }}
                />
              ) : null
            }
            renderCard={(column, order) => (
              <LineBoardWorkOrderCard
                order={order}
                workOrderCardContext={model.workOrderCardContext}
                onOpen={() => onOpenWorkOrder(order, column.key)}
                isAnalyzing={column.key === "backlog" && Boolean(order.id && model.analyzingOrderIds.has(order.id))}
                creditLabel={order.id ? model.creditFailureLabels.get(order.id) : undefined}
                showMergeConfidence={column.key !== "done"}
              />
            )}
          />
          {canCreateWorkOrder ? (
            <div className="absolute right-4 bottom-5 z-20">
              <MobileCreateTaskButton className="size-14 rounded-full shadow-lg" onImported={onOpenWorkOrder} />
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
