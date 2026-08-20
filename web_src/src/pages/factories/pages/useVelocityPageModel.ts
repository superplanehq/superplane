import { useState } from "react";

import type { FactoriesDescribeFactoryVelocityResponse, FactoriesFactoryOnboarding } from "@/api-client";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryVelocity } from "@/hooks/useFactoryVelocity";

import {
  aggregateFactoryVelocityFlow,
  factoryVelocityPeriodLabel,
  formatVelocityYesterdayLabel,
  type FactoryVelocityFlow,
} from "../lib/factoryVelocityFlow";
import type { VelocityData, VelocityDayPoint, VelocityPeriodDays } from "./VelocityLoadedView";

function computeWastePct(merged: number, waste: number): number {
  const total = merged + waste;
  if (total <= 0) return 0;
  return Math.round((waste / total) * 100);
}

export interface VelocityPageModel {
  periodDays: VelocityPeriodDays;
  periodLabel: string;
  setPeriodDays: (days: VelocityPeriodDays) => void;

  /** Whether the workspace has a GitHub integration + repository configured from onboarding. */
  hasRepoConfigured: boolean;

  velocity: {
    data?: VelocityData;
    isLoading: boolean;
    error: unknown;
    refetch: () => void;
    hasPeopleCohort: boolean;
    peopleSearchFailed: boolean;
    repositoryLabel?: string;
  };

  workOrderFlow: {
    flow: FactoryVelocityFlow | null;
    isLoading: boolean;
    error: unknown;
  };
}

function toDayPoint(point: {
  day?: string;
  superplaneMerged?: number;
  peopleMerged?: number;
  waste?: number;
}): VelocityDayPoint {
  return {
    day: point.day ?? "",
    merged: point.superplaneMerged ?? 0,
    waste: point.waste ?? 0,
    peopleMerged: point.peopleMerged ?? 0,
    superplaneMerged: point.superplaneMerged ?? 0,
  };
}

function toVelocityData(response: FactoriesDescribeFactoryVelocityResponse, dateLabel: string): VelocityData {
  const yesterday = response.yesterday ?? {};
  const totals = response.totals ?? {};
  const points = response.points ?? [];

  return {
    yesterday: {
      dateLabel,
      merged: yesterday.superplaneMerged ?? 0,
      waste: yesterday.waste ?? 0,
      wastePct: computeWastePct(yesterday.superplaneMerged ?? 0, yesterday.waste ?? 0),
    },
    totals: {
      merged: totals.superplaneMerged ?? 0,
      waste: totals.waste ?? 0,
      wastePct: totals.wastePct ?? 0,
      superplaneMerged: totals.superplaneMerged ?? 0,
      peopleMerged: totals.peopleMerged ?? 0,
      superplaneSharePct: totals.superplaneSharePct ?? 0,
    },
    points: points.map(toDayPoint),
  };
}

export function useVelocityPageModel(
  organizationId: string,
  factoryId: string,
  onboarding: FactoriesFactoryOnboarding | null | undefined,
): VelocityPageModel {
  const [periodDays, setPeriodDays] = useState<VelocityPeriodDays>(7);

  // The velocity chart always uses the workspace's own GitHub integration and
  // repository (set during onboarding) rather than letting the user pick one
  // per-visit — there is exactly one "right" repo for a given workspace.
  const integrationId = onboarding?.vcsIntegrationId ?? "";
  const repository = onboarding?.appRepository ?? "";
  const hasRepoConfigured = Boolean(integrationId && repository);

  const {
    data: velocityResponse,
    isLoading: velocityLoading,
    isFetching: velocityFetching,
    error: velocityError,
    refetch: refetchVelocity,
  } = useFactoryVelocity(organizationId, factoryId, {
    periodDays,
    integrationId: integrationId || undefined,
    repository: repository || undefined,
  });

  const {
    data: workOrders = [],
    isLoading: workOrdersLoading,
    isFetching: workOrdersFetching,
    error: workOrdersError,
  } = useFactoryWorkOrders(organizationId, factoryId);

  const isVelocityLoading = velocityLoading || (velocityFetching && !velocityResponse);
  const isWorkOrdersLoading = workOrdersLoading || (workOrdersFetching && workOrders.length === 0);

  const hasPeopleCohort = Boolean(velocityResponse?.hasPeopleCohort && hasRepoConfigured);

  const velocityData = velocityResponse
    ? toVelocityData(velocityResponse, formatVelocityYesterdayLabel(velocityResponse.yesterday?.date))
    : undefined;
  const workOrderFlow = isWorkOrdersLoading ? null : aggregateFactoryVelocityFlow(workOrders, periodDays);

  return {
    periodDays,
    periodLabel: factoryVelocityPeriodLabel(periodDays),
    setPeriodDays,

    hasRepoConfigured,

    velocity: {
      data: velocityData,
      isLoading: isVelocityLoading,
      error: velocityError,
      refetch: () => void refetchVelocity(),
      hasPeopleCohort,
      peopleSearchFailed: Boolean(velocityResponse?.peopleSearchFailed),
      repositoryLabel: velocityResponse?.repository || repository || undefined,
    },

    workOrderFlow: {
      flow: workOrderFlow,
      isLoading: isWorkOrdersLoading,
      error: workOrdersError,
    },
  };
}
