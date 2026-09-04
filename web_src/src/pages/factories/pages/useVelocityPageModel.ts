import { useEffect, useMemo, useRef, useState } from "react";

import type { FactoriesFactory } from "@/api-client";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryVelocity, useSyncFactoryVelocity } from "@/hooks/useFactoryVelocity";

import {
  aggregateFactoryVelocityFlow,
  factoryVelocityPeriodLabel,
  type FactoryVelocityFlow,
} from "../lib/factoryVelocityFlow";
import {
  hasVelocityOutput,
  toVelocityReport,
  type VelocityPeriodDays,
  type VelocityPerson,
  type VelocityReport,
} from "../lib/factoryVelocityReport";
import {
  PEOPLE_SORT_DEFAULT_DIRECTION,
  PEOPLE_SORT_DEFAULT_KEY,
  nextPeopleOffset,
  peoplePageSizeForOffset,
  type PeopleSortDirection,
  type PeopleSortKey,
} from "../lib/velocityPeopleSort";
import type { VelocityComparison } from "./velocityCards";

const MS_PER_DAY = 24 * 60 * 60 * 1000;

export interface VelocityPageModel {
  periodDays: VelocityPeriodDays;
  periodLabel: string;
  setPeriodDays: (days: VelocityPeriodDays) => void;

  /** GitHub integration selected during workspace setup. */
  integrationId: string;
  /** App repository selected during workspace setup, as `owner/repo`. */
  repository: string;

  velocity: {
    report?: VelocityReport;
    isLoading: boolean;
    error: unknown;
    refetch: () => void;
    /** True when the first repository sync has not stored any merges yet. */
    peopleSyncPending: boolean;
    /** True when the window holds no merges, no waste, and no spend. */
    isEmpty: boolean;
    /** When the background sync last stored repository merges. */
    syncedAt?: Date;
  };

  /**
   * The People table's rows, sorting, and paging. Sorting happens on the
   * backend, so changing the column or its direction refetches from the
   * first page. "Show more" appends the next page to what is already shown.
   */
  people: {
    /** Rows fetched so far, in backend-sorted order, ranked from the top. */
    list: VelocityPerson[];
    /** Total people with activity in the window, across every page. */
    total: number;
    sortKey: PeopleSortKey;
    sortDirection: PeopleSortDirection;
    /** Sorts by `key`, toggling direction on the already-active column. */
    onSort: (key: PeopleSortKey) => void;
    canLoadMore: boolean;
    isLoadingMore: boolean;
    loadMore: () => void;
  };

  /** Asks for a fresh read of the repository merges the report is built from. */
  sync: {
    start: () => void;
    isSyncing: boolean;
    /** True when the workspace has no repository to sync. */
    isUnavailable: boolean;
  };

  taskTime: {
    flow: FactoryVelocityFlow | null;
    isLoading: boolean;
    error: unknown;
  };

  /** Deltas against the previous window, for the metrics that have a baseline. */
  comparison: VelocityComparison;
}

export function useVelocityPageModel(
  organizationId: string,
  factoryId: string,
  onboarding: FactoriesFactory["onboarding"],
): VelocityPageModel {
  const [periodDays, setPeriodDays] = useState<VelocityPeriodDays>(14);

  const integrationId = onboarding?.vcsIntegrationId?.trim() ?? "";
  const repository = onboarding?.appRepository?.trim() ?? "";

  const peopleSort = usePeopleSortAndPaging(periodDays, repository);

  const {
    data: velocityResponse,
    isLoading: velocityLoading,
    isFetching: velocityFetching,
    isPlaceholderData: holdsPreviousReport,
    error: velocityError,
    refetch: refetchVelocity,
  } = useFactoryVelocity(organizationId, factoryId, {
    periodDays,
    repository: repository || undefined,
    peopleSort: peopleSort.sortKey,
    peopleSortDirection: peopleSort.sortDirection,
    peopleOffset: peopleSort.offset,
    peoplePageSize: peoplePageSizeForOffset(peopleSort.offset),
  });

  const syncVelocity = useSyncFactoryVelocity(organizationId, factoryId);

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const isVelocityLoading = velocityLoading || (velocityFetching && !velocityResponse);
  const isWorkOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);

  const report = useMemo(() => (velocityResponse ? toVelocityReport(velocityResponse) : undefined), [velocityResponse]);

  const people = useAccumulatedPeople(peopleSort.resetKey, peopleSort.offset, report, Boolean(holdsPreviousReport));
  const isLoadingMorePeople = velocityFetching && peopleSort.offset > 0;

  // Task time is measured from work orders, which carry the execution
  // timestamps the velocity API does not report.
  const flow = useMemo(
    () => (isWorkOrdersLoading ? null : aggregateFactoryVelocityFlow(workOrders, periodDays)),
    [isWorkOrdersLoading, workOrders, periodDays],
  );
  const previousFlow = useMemo(
    () =>
      isWorkOrdersLoading
        ? null
        : aggregateFactoryVelocityFlow(workOrders, periodDays, Date.now() - periodDays * MS_PER_DAY),
    [isWorkOrdersLoading, workOrders, periodDays],
  );

  const comparison = useMemo(() => buildComparison(report, flow, previousFlow), [report, flow, previousFlow]);

  return {
    periodDays,
    periodLabel: factoryVelocityPeriodLabel(periodDays),
    setPeriodDays,

    integrationId,
    repository,

    velocity: {
      report,
      isLoading: isVelocityLoading,
      error: velocityError,
      refetch: () => void refetchVelocity(),
      peopleSyncPending: Boolean(velocityResponse?.peopleSyncPending),
      isEmpty: report ? !hasVelocityOutput(report) : false,
      syncedAt: report?.peopleSyncedAt,
    },

    people: {
      list: people.list,
      total: people.total,
      sortKey: peopleSort.sortKey,
      sortDirection: peopleSort.sortDirection,
      onSort: peopleSort.onSort,
      canLoadMore: people.canLoadMore && !Boolean(holdsPreviousReport),
      isLoadingMore: isLoadingMorePeople,
      loadMore: () => {
        if (isLoadingMorePeople || holdsPreviousReport || !people.canLoadMore) return;
        peopleSort.loadMore();
      },
    },

    sync: {
      start: () => void syncVelocity.mutate(),
      isSyncing: syncVelocity.isPending,
      isUnavailable: !repository,
    },

    taskTime: {
      flow,
      isLoading: isWorkOrdersLoading,
      error: workOrdersError,
    },

    comparison,
  };
}

/**
 * Owns the People table's sort key, direction, and page offset.
 *
 * Changing the sort, the period, or the repository starts a new cohort: the
 * offset resets to the first page. This mirrors React's "adjust state during
 * render" pattern rather than an effect, so the reset lands before the
 * `useFactoryVelocity` call that reads `offset` for the same render.
 */
function usePeopleSortAndPaging(periodDays: VelocityPeriodDays, repository: string) {
  const [sortKey, setSortKey] = useState<PeopleSortKey>(PEOPLE_SORT_DEFAULT_KEY);
  const [sortDirection, setSortDirection] = useState<PeopleSortDirection>(PEOPLE_SORT_DEFAULT_DIRECTION);
  const [offset, setOffset] = useState(0);

  const resetKey = `${periodDays}|${repository}|${sortKey}|${sortDirection}`;
  const [appliedResetKey, setAppliedResetKey] = useState(resetKey);
  if (appliedResetKey !== resetKey) {
    setAppliedResetKey(resetKey);
    setOffset(0);
  }

  const onSort = (key: PeopleSortKey) => {
    if (key === sortKey) {
      setSortDirection((direction) => (direction === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(key);
    setSortDirection(PEOPLE_SORT_DEFAULT_DIRECTION);
  };

  const loadMore = () => setOffset((current) => nextPeopleOffset(current));

  return { sortKey, sortDirection, offset, resetKey, onSort, loadMore };
}

/**
 * Accumulates the People pages fetched so far. A ref (not state) tracks which
 * page was last applied, so a background refetch of the same page replaces it
 * in place instead of duplicating it, while a genuinely new page (a fresh
 * offset, sort, period, or repository) is appended, or replaces everything
 * when it is the first page.
 *
 * While the next page loads, the query still answers with the report that is
 * on screen. That report belongs to the previous offset, so `holdsPreviousReport`
 * keeps it out of the list until the requested page arrives.
 */
function useAccumulatedPeople(
  resetKey: string,
  offset: number,
  report: VelocityReport | undefined,
  holdsPreviousReport: boolean,
) {
  const [list, setList] = useState<VelocityPerson[]>([]);
  const appliedPageRef = useRef<string | null>(null);

  useEffect(() => {
    if (!report || holdsPreviousReport) return;
    const pageKey = `${resetKey}|${offset}`;
    if (appliedPageRef.current === pageKey) return;
    appliedPageRef.current = pageKey;
    setList((prev) => (offset === 0 ? report.people : [...prev, ...report.people]));
  }, [report, resetKey, offset, holdsPreviousReport]);

  const total = report?.peopleTotal ?? list.length;
  const canLoadMore = list.length < total;

  return { list, total, canLoadMore };
}

/**
 * Deltas are reported per metric, because their baselines differ: task counts
 * and spend need the previous API window, while cycle time is measured from the
 * work orders themselves. A metric with no baseline shows no chip at all, which
 * is honest about a workspace that only started last week.
 */
function buildComparison(
  report: VelocityReport | undefined,
  flow: FactoryVelocityFlow | null,
  previousFlow: FactoryVelocityFlow | null,
): VelocityComparison {
  const comparison: VelocityComparison = {};
  if (!report) return comparison;

  const previous = report.previous;
  if (previous) {
    comparison.tasksClosed = report.totals.tasksClosed - previous.tasksClosed;
    comparison.taskWasteRate = report.totals.taskWasteRate - previous.taskWasteRate;
    comparison.costPerTask = round(report.totals.costPerTask - previous.costPerTask, 2);
  }

  if (flow && previousFlow && flow.sampleSize > 0 && previousFlow.sampleSize > 0) {
    comparison.cycleHours = round(flow.medianCycleHours - previousFlow.medianCycleHours, 1);
  }

  return comparison;
}

function round(value: number, decimals: number): number {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}
