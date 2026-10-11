import type {
  FactoriesFactoryIntake,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderSummary,
} from "@/api-client";
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
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Navigate, useLocation, useNavigate, useParams } from "react-router";

import { backlogAnalysisCreditLabels } from "../lib/backlogAnalysis";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { useLineBoardColumnColorViewPreference } from "../lib/lineBoardColumnColorViewPreference";
import {
  factoryHomePath,
  factoryLineDetailPath,
  intakeIdFromSearch,
  isIntakeSearchOpen,
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
import { PhaseGlyph } from "../pages/linePhaseGlyph";
import { usePRFeedbackWorkOrderAttention } from "../pages/useWorkOrderPRFeedbackRunHref";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { MobileColumn, type ColumnPaging } from "./MobileBoardColumn";
import { MobileBoardHeader } from "./MobileBoardHeader";
import { MobileBoardSettings } from "./MobileBoardSettings";
import { activeColumnIndex, buildMobileBoardColumns, type MobileBoardColumn } from "./mobileBoardColumns";

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
      onOpenWorkOrder={(order) => {
        const number = canonicalWorkOrderNumber(order);
        if (number) {
          navigate(workOrderDetailPath(organizationId, routeSegment, number), {
            state: { peekOrder: order, lineId: selectedLine.id },
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
  onOpenWorkOrder: (order: FactoriesWorkOrderSummary) => void;
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
  const lines = factory?.lines ?? [];
  const { search } = useLocation();
  const intakeSettingsOpen =
    isIntakeSearchOpen(search) && configuredIntakes.some((intake) => intake.intakeId === intakeIdFromSearch(search));

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
      />
      {!intakeSettingsOpen ? (
        <>
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
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={routeSegment}
            canManageClosedStatus={canConfigureFactory}
          />
          <MobileColumnCarousel
            onImported={onOpenWorkOrder}
            columns={model.columns}
            cardsPending={model.cardsPending}
            laneClassName={(column) =>
              lineBoardColumnLaneProps(columnColors[column.key] ?? null, colorView, { mutedFallback: true })
            }
            paging={model.paging}
            renderCard={(column, order) => (
              <LineBoardWorkOrderCard
                order={order}
                workOrderCardContext={model.workOrderCardContext}
                onOpen={() => onOpenWorkOrder(order)}
                isAnalyzing={column.key === "backlog" && Boolean(order.id && model.analyzingOrderIds.has(order.id))}
                creditLabel={order.id ? model.creditFailureLabels.get(order.id) : undefined}
                showMergeConfidence={column.key !== "done"}
              />
            )}
          />
        </>
      ) : null}
    </div>
  );
}

/**
 * Horizontal snap carousel. Each column fills the screen width and scrolls
 * on its own. Tabs above mirror the swipe position and jump on tap.
 */
function MobileColumnCarousel({
  columns,
  cardsPending,
  laneClassName,
  paging,
  renderCard,
  onImported,
}: {
  columns: MobileBoardColumn[];
  cardsPending: boolean;
  laneClassName: (column: MobileBoardColumn) => { className?: string; surfaceClassName?: string };
  paging: Record<MobileBoardColumn["paging"], ColumnPaging>;
  renderCard: (column: MobileBoardColumn, order: FactoriesWorkOrder) => ReactNode;
  onImported: (order: FactoriesWorkOrder) => void;
}) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [activeIndex, setActiveIndex] = useState(0);

  const syncActiveFromScroll = useCallback(() => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    setActiveIndex(activeColumnIndex(track.scrollLeft, track.clientWidth, columns.length));
  }, [columns.length]);

  const scrollToColumn = useCallback((index: number) => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    track.scrollTo({ left: index * track.clientWidth, behavior: "smooth" });
    setActiveIndex(index);
  }, []);

  useEffect(() => {
    if (activeIndex > columns.length - 1) {
      setActiveIndex(Math.max(columns.length - 1, 0));
    }
  }, [activeIndex, columns.length]);

  return (
    <>
      <MobileColumnTabs columns={columns} activeIndex={activeIndex} onSelect={scrollToColumn} />
      <div
        ref={trackRef}
        onScroll={syncActiveFromScroll}
        className="flex min-h-0 flex-1 snap-x snap-mandatory overflow-x-auto overflow-y-hidden [scrollbar-width:none]"
        data-testid="mobile-board-track"
      >
        {columns.map((column, index) => (
          <MobileColumn
            key={column.key}
            column={column}
            hidden={index !== activeIndex}
            cardsPending={cardsPending}
            lane={laneClassName(column)}
            paging={paging[column.paging]}
            renderCard={(order) => renderCard(column, order)}
            onImported={onImported}
          />
        ))}
      </div>
    </>
  );
}

function MobileColumnTabs({
  columns,
  activeIndex,
  onSelect,
}: {
  columns: MobileBoardColumn[];
  activeIndex: number;
  onSelect: (index: number) => void;
}) {
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  useEffect(() => {
    tabRefs.current[activeIndex]?.scrollIntoView({ inline: "center", block: "nearest", behavior: "smooth" });
  }, [activeIndex]);

  return (
    <div
      role="tablist"
      aria-label="Board columns"
      className="flex shrink-0 gap-1 overflow-x-auto border-b border-border px-4 [scrollbar-width:none]"
      data-testid="mobile-board-tabs"
    >
      {columns.map((column, index) => {
        const active = index === activeIndex;
        return (
          <button
            key={column.key}
            ref={(element) => {
              tabRefs.current[index] = element;
            }}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onSelect(index)}
            data-testid={`mobile-board-tab-${column.key}`}
            className={cn(
              "flex h-12 shrink-0 items-center gap-1.5 border-b-2 px-2.5 text-[13px] font-medium tracking-[-0.01em] transition-colors",
              active ? "border-foreground text-foreground" : "border-transparent text-muted-foreground",
            )}
          >
            {column.glyph ? <PhaseGlyph kind={column.glyph} className="size-3" /> : null}
            <span className="truncate">{column.title}</span>
            <span className="tabular-nums text-[12px] text-muted-foreground">
              {column.totalCount ?? column.cards.length}
            </span>
          </button>
        );
      })}
    </div>
  );
}
