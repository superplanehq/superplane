import { useMemo, useState } from "react";

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
  type VelocityReport,
} from "../lib/factoryVelocityReport";
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

  const {
    data: velocityResponse,
    isLoading: velocityLoading,
    isFetching: velocityFetching,
    error: velocityError,
    refetch: refetchVelocity,
  } = useFactoryVelocity(organizationId, factoryId, {
    periodDays,
    repository: repository || undefined,
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
 * Deltas are reported per metric, because their baselines differ: merges and
 * spend need the previous API window, while cycle time needs closed tasks in
 * that window. A metric with no baseline shows no chip at all, which is
 * honest about a workspace that only started last week.
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
    comparison.merged = report.totals.merged - previous.merged;
    comparison.wasteRate = report.totals.wasteRate - previous.wasteRate;
    comparison.costPerMerge = round(report.totals.costPerMerge - previous.costPerMerge, 2);
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
