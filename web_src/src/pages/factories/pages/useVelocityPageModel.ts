import { useMemo, useState } from "react";

import type { FactoriesDescribeFactoryVelocityResponse, OrganizationsIntegration } from "@/api-client";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryVelocity } from "@/hooks/useFactoryVelocity";
import { useConnectedIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";

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

  githubIntegrations: OrganizationsIntegration[];
  integrationId: string;
  setIntegrationId: (integrationId: string) => void;

  repositoryOptions: string[];
  repositoriesLoading: boolean;
  repository: string;
  setRepository: (repository: string) => void;

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

export function useVelocityPageModel(organizationId: string, factoryId: string): VelocityPageModel {
  const [periodDays, setPeriodDays] = useState<VelocityPeriodDays>(7);
  const [integrationId, setIntegrationId] = useState<string>("");
  const [repository, setRepository] = useState<string>("");

  const { data: connectedIntegrations = [] } = useConnectedIntegrations(organizationId);
  const githubIntegrations = useMemo(
    () =>
      connectedIntegrations.filter(
        (integration) => integration.metadata?.integrationName === "github" && integration.status?.state === "ready",
      ),
    [connectedIntegrations],
  );

  const effectiveIntegrationId = useMemo(() => {
    if (integrationId) return integrationId;
    if (githubIntegrations.length === 1) {
      return githubIntegrations[0].metadata?.id ?? "";
    }
    return "";
  }, [integrationId, githubIntegrations]);

  const { data: repositoryResources = [], isLoading: repositoriesLoading } = useIntegrationResources(
    organizationId,
    effectiveIntegrationId,
    "repository",
  );

  const repositoryOptions = useMemo(
    () =>
      repositoryResources
        .map((resource) => resource.name?.trim() ?? "")
        .filter((name): name is string => name.length > 0),
    [repositoryResources],
  );

  const {
    data: velocityResponse,
    isLoading: velocityLoading,
    isFetching: velocityFetching,
    error: velocityError,
    refetch: refetchVelocity,
  } = useFactoryVelocity(organizationId, factoryId, {
    periodDays,
    integrationId: effectiveIntegrationId || undefined,
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

  const hasRepoContext = Boolean(effectiveIntegrationId && repository);
  const hasPeopleCohort = Boolean(velocityResponse?.hasPeopleCohort && hasRepoContext);

  const velocityData = velocityResponse
    ? toVelocityData(velocityResponse, formatVelocityYesterdayLabel(velocityResponse.yesterday?.date))
    : undefined;
  const workOrderFlow = isWorkOrdersLoading ? null : aggregateFactoryVelocityFlow(workOrders, periodDays);

  return {
    periodDays,
    periodLabel: factoryVelocityPeriodLabel(periodDays),
    setPeriodDays,

    githubIntegrations,
    integrationId: effectiveIntegrationId,
    setIntegrationId: (value: string) => {
      setIntegrationId(value);
      setRepository("");
    },

    repositoryOptions,
    repositoriesLoading,
    repository,
    setRepository,

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
