import type { CanvasesCanvasRun, FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Input } from "@/components/ui/input";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { useFactoryPullRequests, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { useAuxiliarySidebarWidth } from "@/stores/useAuxiliarySidebarWidth";
import { AlertCircle, Loader2, Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";

import { findWorkOrderForAutomationRun } from "../../lib/factoryAutomationStatus";
import { useOptionalFactoriesLayout } from "../../layout/factoriesLayoutContext";
import type { WorkOrderCardContext } from "../../workOrders/WorkOrderCard";
import { usePRFeedbackWorkOrderAttention } from "../useWorkOrderPRFeedbackRunHref";
import { FactoryAutomationRunItem } from "./FactoryAutomationRunItem";
import { factoryAutomationRunMatchesQuery } from "./factoryAutomationRunSearch";

export const FACTORY_AUTOMATION_RUNS_SIDEBAR_TEST_ID = "factory-automation-runs-sidebar";
const WIDTH_STORAGE_KEY = "factory-automation-runs-sidebar-width";
const DEFAULT_WIDTH = 320;

interface FactoryAutomationRunsSidebarProps {
  canvasId: string;
  organizationId?: string;
  factoryId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  selectedRunId: string | null;
  onSelectRun: (runId: string | null) => void;
}

/** Task-card run list for factory line automations. Not the standalone canvas sidebar. */
export function FactoryAutomationRunsSidebar({
  canvasId,
  organizationId: organizationIdProp,
  factoryId: factoryIdProp,
  runHrefFor,
  selectedRunId,
  onSelectRun,
}: FactoryAutomationRunsSidebarProps) {
  const layout = useOptionalFactoriesLayout();
  const organizationId = layout?.organizationId ?? organizationIdProp ?? "";
  const factoryId = layout?.factoryId ?? factoryIdProp ?? "";
  const factoryKey = layout?.factoryKey ?? "";
  const factory = layout?.factory ?? null;
  const { sidebarRef, width, isResizing, handleMouseDown } = useAuxiliarySidebarWidth(
    true,
    WIDTH_STORAGE_KEY,
    DEFAULT_WIDTH,
  );

  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, Boolean(canvasId));
  const runs = useMemo(() => runsQuery.data?.pages.flatMap((page) => page?.runs ?? []) ?? [], [runsQuery.data]);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const { data: pullRequests = [] } = useFactoryPullRequests(organizationId, factoryId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const attention = usePRFeedbackWorkOrderAttention(pullRequests);
  const [searchQuery, setSearchQuery] = useState("");

  const workOrderCardContext = useMemo(() => {
    if (!organizationId || !factoryKey) {
      return null;
    }
    return {
      organizationId,
      factoryId,
      factoryKey,
      factoryLines: factory?.lines ?? [],
      canDispatch: Boolean(layout),
      canAssign: Boolean(layout),
      addressingFeedbackOrderIds: attention.addressingFeedbackOrderIds,
      addressingFeedbackLabels: attention.addressingFeedbackLabels,
      waitingOnChecksOrderIds: attention.waitingOnChecksOrderIds,
      checksPassedOrderIds: attention.checksPassedOrderIds,
      fixesPausedOrderIds: attention.fixesPausedOrderIds,
      pullRequests,
      ...cardActions,
    };
  }, [attention, cardActions, factory?.lines, factoryId, factoryKey, layout, organizationId, pullRequests]);

  const visibleRuns = useMemo(
    () =>
      runs.filter((run) => {
        if (!run.id) {
          return false;
        }
        const order = findWorkOrderForAutomationRun(workOrders, run.id, run);
        return factoryAutomationRunMatchesQuery(searchQuery, run, order);
      }),
    [runs, searchQuery, workOrders],
  );

  const scrollRef = useRef<HTMLDivElement>(null);
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: runsQuery.hasNextPage,
    isLoading: runsQuery.isFetchingNextPage,
    onLoadMore: () => void runsQuery.fetchNextPage(),
  });

  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [visibleRuns.length, loadMoreIfNeeded]);

  const handleScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      loadMoreIfNeeded(event.currentTarget);
    },
    [loadMoreIfNeeded],
  );

  return (
    <aside
      ref={sidebarRef}
      data-testid={FACTORY_AUTOMATION_RUNS_SIDEBAR_TEST_ID}
      className={cn(
        "relative z-30 flex h-full min-w-0 shrink-0 flex-col border-r bg-background",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width, maxWidth: width }}
    >
      <FactoryAutomationRunsToolbar searchQuery={searchQuery} onSearchQueryChange={setSearchQuery} />
      <div
        ref={scrollRef}
        className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto"
        data-testid="factory-automation-runs-scroll"
        onScroll={handleScroll}
      >
        <FactoryAutomationRunsList
          runs={visibleRuns}
          allRunsCount={runs.length}
          workOrders={workOrders}
          factory={factory}
          workOrderCardContext={workOrderCardContext}
          selectedRunId={selectedRunId}
          onSelectRun={onSelectRun}
          runHrefFor={runHrefFor}
          isLoading={runsQuery.isPending}
          isError={runsQuery.isError}
          onRetry={() => void runsQuery.refetch()}
        />
      </div>
      <FactoryAutomationRunsResizeHandle isResizing={isResizing} onMouseDown={handleMouseDown} />
    </aside>
  );
}

function FactoryAutomationRunsToolbar({
  searchQuery,
  onSearchQueryChange,
}: {
  searchQuery: string;
  onSearchQueryChange: (query: string) => void;
}) {
  return (
    <div
      className="flex h-10 shrink-0 items-center gap-2 border-b px-3 pr-1.5"
      data-testid="factory-automation-runs-toolbar"
    >
      <span className="shrink-0 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Runs</span>
      <div className="relative min-w-0 flex-1">
        <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="text"
          value={searchQuery}
          onChange={(event) => onSearchQueryChange(event.target.value)}
          aria-label="Search runs"
          placeholder="Search…"
          className="h-7 rounded-none border-0 bg-transparent pr-2 pl-7 text-[13px] shadow-none"
        />
      </div>
    </div>
  );
}

function FactoryAutomationRunsList({
  runs,
  allRunsCount,
  workOrders,
  factory,
  workOrderCardContext,
  selectedRunId,
  onSelectRun,
  runHrefFor,
  isLoading,
  isError,
  onRetry,
}: {
  runs: CanvasesCanvasRun[];
  allRunsCount: number;
  workOrders: FactoriesWorkOrder[];
  factory: FactoriesFactory | null;
  workOrderCardContext: WorkOrderCardContext | null;
  selectedRunId: string | null;
  onSelectRun: (runId: string | null) => void;
  runHrefFor?: RunsSidebarHrefForRun;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}) {
  if (isError && allRunsCount === 0) {
    return (
      <div
        role="alert"
        className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-muted-foreground"
      >
        <AlertCircle className="h-5 w-5 text-destructive" aria-hidden />
        <span>The runs failed to load.</span>
        <button type="button" onClick={onRetry} className="text-[11px] text-sky-600 hover:text-sky-800">
          Try again
        </button>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (allRunsCount === 0) {
    return <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">No runs</p>;
  }

  if (runs.length === 0) {
    return <p className="px-3 py-6 text-center text-xs text-muted-foreground">No runs match your search.</p>;
  }

  return (
    <>
      {runs.map((run) => (
        <FactoryAutomationRunItem
          key={run.id}
          run={run}
          workOrders={workOrders}
          factory={factory}
          workOrderCardContext={workOrderCardContext}
          runHref={runHrefFor?.(run.id ?? "") ?? "#"}
          isSelected={run.id === selectedRunId}
          onSelect={onSelectRun}
        />
      ))}
    </>
  );
}

function FactoryAutomationRunsResizeHandle({
  isResizing,
  onMouseDown,
}: {
  isResizing: boolean;
  onMouseDown: (event: React.MouseEvent) => void;
}) {
  return (
    <div
      onMouseDown={onMouseDown}
      className="group absolute top-0 right-0 bottom-0 z-50 w-4 cursor-col-resize bg-transparent"
      style={{ marginRight: "-8px" }}
      data-testid="factory-automation-runs-sidebar-resize-handle"
    >
      <div
        aria-hidden
        className={cn(
          "pointer-events-none absolute top-0 bottom-0 left-[calc(50%+1px)] w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-500/50",
          isResizing && "bg-slate-950/50 dark:bg-gray-500/50",
        )}
      />
    </div>
  );
}
