import {
  factoriesCreateFactoryIntake,
  factoriesDeleteFactoryIntake,
  factoriesImportFactoryIntakeItem,
  factoriesListFactoryIntakeRuns,
  factoriesListFactoryIntakes,
  factoriesRefreshBacklog,
  factoriesSearchDependabotIntakeSetupItems,
  factoriesSearchFactoryIntakeItems,
  factoriesUpdateFactoryIntake,
} from "@/api-client";
import type {
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeItem,
  FactoriesFactoryIntakeRun,
  FactoriesFactoryIntakeSource,
  FactoriesWorkOrder,
  FactoriesFactoryIntakeSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";

import { factoryAppsKey, factoryQueryKeys } from "./useFactoryData";
import { useCanvasWebsocket } from "./useCanvasWebsocket";
import { applyWorkOrderToListCaches } from "./workOrderListCache";

const INTAKE_RUN_REFRESH_BATCH_MS = 250;

const factoryIntakeQueryKeys = {
  list: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId, "intakes"] as const,
  runs: (organizationId: string, factoryId: string, intakeId: string) =>
    ["factories", organizationId, factoryId, "intakes", intakeId, "runs"] as const,
  items: (organizationId: string, factoryId: string, intakeId: string, query: string, limit: number) =>
    ["factories", organizationId, factoryId, "intakes", intakeId, "items", query, limit] as const,
  dependabotSetupItems: (organizationId: string, factoryId: string, severities: string, limit: number) =>
    ["factories", organizationId, factoryId, "dependabot-setup-items", severities, limit] as const,
};

export function factoryIntakesKey(organizationId: string, factoryId: string) {
  return factoryIntakeQueryKeys.list(organizationId, factoryId);
}

export function factoryIntakeRunsKey(organizationId: string, factoryId: string, intakeId: string) {
  return factoryIntakeQueryKeys.runs(organizationId, factoryId, intakeId);
}

export async function fetchFactoryIntakes(
  organizationId: string,
  factoryId: string,
): Promise<FactoriesFactoryIntake[]> {
  const response = await factoriesListFactoryIntakes(
    withOrganizationHeader({
      organizationId,
      path: { factoryId },
    }),
  );
  return response.data?.intakes ?? [];
}

export function useFactoryIntakes(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryIntakesKey(organizationId, factoryId),
    queryFn: () => fetchFactoryIntakes(organizationId, factoryId),
    enabled: Boolean(organizationId && factoryId),
  });
}

export function useFactoryIntakeRuns(
  organizationId: string,
  factoryId: string,
  intakeId: string | undefined,
  enabled = true,
) {
  return useQuery({
    queryKey: factoryIntakeRunsKey(organizationId, factoryId, intakeId ?? ""),
    queryFn: async (): Promise<FactoriesFactoryIntakeRun[]> => {
      const response = await factoriesListFactoryIntakeRuns(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId: intakeId ?? "" },
        }),
      );
      return response.data?.runs ?? [];
    },
    enabled: Boolean(organizationId && factoryId && intakeId) && enabled,
  });
}

/** Refresh the derived intake-run view after its canvas changes. */
export function useFactoryIntakeRunsWebsocket({
  organizationId,
  factoryId,
  intakeId,
  canvasId,
}: {
  organizationId: string;
  factoryId: string;
  intakeId: string | undefined;
  canvasId: string | undefined;
}): void {
  const queryClient = useQueryClient();
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const scheduleRefresh = useCallback(() => {
    clearTimeout(refreshTimer.current);
    refreshTimer.current = setTimeout(() => {
      if (!intakeId) {
        return;
      }
      void queryClient.invalidateQueries({
        queryKey: factoryIntakeRunsKey(organizationId, factoryId, intakeId),
      });
    }, INTAKE_RUN_REFRESH_BATCH_MS);
  }, [factoryId, intakeId, organizationId, queryClient]);

  useEffect(
    () => () => {
      clearTimeout(refreshTimer.current);
    },
    [],
  );

  useCanvasWebsocket({
    canvasId: canvasId ?? "",
    organizationId,
    processRuntimeEvents: true,
    enabled: Boolean(organizationId && factoryId && intakeId && canvasId),
    onRunEvent: scheduleRefresh,
    onExecutionEvent: scheduleRefresh,
    onConnectionOpen: scheduleRefresh,
  });
}

export function useCreateFactoryIntake(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      source: FactoriesFactoryIntakeSource;
      name?: string;
      confidencePct?: number;
      integrationId?: string;
      resourceId?: string;
      skipInitialImport?: boolean;
      settings?: FactoriesFactoryIntakeSettings;
    }) => {
      const response = await factoriesCreateFactoryIntake(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            source: input.source,
            name: input.name,
            confidencePct: input.confidencePct,
            integrationId: input.integrationId,
            resourceId: input.resourceId,
            skipInitialImport: input.skipInitialImport,
            settings: input.settings,
          },
        }),
      );
      if (!response.data?.intake) {
        throw new Error("Failed to create intake");
      }
      return response.data.intake;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryIntakesKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      // A new intake seeds the newest items of its source, so the Backlog
      // already holds tasks the cached list does not know about.
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
    },
  });
}

export function useDeleteFactoryIntake(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (intakeId: string) => {
      await factoriesDeleteFactoryIntake(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId },
        }),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryIntakesKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
    },
  });
}

export function useUpdateFactoryIntake(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      intakeId: string;
      name?: string;
      settings?: FactoriesFactoryIntakeSettings;
      paused?: boolean;
      integrationId?: string;
      resourceId?: string;
    }) => {
      const response = await factoriesUpdateFactoryIntake(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId: input.intakeId },
          body: {
            name: input.name,
            settings: input.settings,
            paused: input.paused,
            integrationId: input.integrationId,
            resourceId: input.resourceId,
          },
        }),
      );
      if (!response.data?.intake) {
        throw new Error("Failed to update intake");
      }
      return response.data.intake;
    },
    onSuccess: (_intake, variables) => {
      void queryClient.invalidateQueries({ queryKey: factoryIntakesKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryIntakeQueryKeys.runs(organizationId, factoryId, variables.intakeId),
      });
    },
  });
}

export function useSearchDependabotIntakeSetupItems({
  organizationId,
  factoryId,
  dependabotSeverities,
  enabled = true,
  limit = 50,
}: {
  organizationId: string;
  factoryId: string;
  dependabotSeverities: string[];
  enabled?: boolean;
  limit?: number;
}) {
  const severitiesKey = dependabotSeverities.join(",");
  return useQuery({
    queryKey: factoryIntakeQueryKeys.dependabotSetupItems(organizationId, factoryId, severitiesKey, limit),
    queryFn: async (): Promise<FactoriesFactoryIntakeItem[]> => {
      const response = await factoriesSearchDependabotIntakeSetupItems(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { dependabotSeverities, limit },
        }),
      );
      return response.data?.items ?? [];
    },
    enabled: Boolean(organizationId && factoryId) && enabled,
  });
}

export function useSearchFactoryIntakeItems({
  organizationId,
  factoryId,
  intakeId,
  query,
  enabled = true,
  limit = 5,
}: {
  organizationId: string;
  factoryId: string;
  intakeId: string | null;
  query: string;
  enabled?: boolean;
  limit?: number;
}) {
  const scopedIntakeId = intakeId ?? "";
  return useQuery({
    queryKey: factoryIntakeQueryKeys.items(organizationId, factoryId, scopedIntakeId, query, limit),
    queryFn: async (): Promise<FactoriesFactoryIntakeItem[]> => {
      const response = await factoriesSearchFactoryIntakeItems(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId: scopedIntakeId },
          query: { query, limit },
        }),
      );
      return response.data?.items ?? [];
    },
    enabled: Boolean(organizationId && factoryId && intakeId) && enabled,
    placeholderData: (previousData, previousQuery) => {
      if (previousQuery?.queryKey[4] !== scopedIntakeId) {
        return undefined;
      }
      if (previousQuery?.queryKey[6] !== query) {
        return undefined;
      }
      return previousData;
    },
  });
}

export function useImportFactoryIntakeItem(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { intakeId: string; itemId: string }): Promise<FactoriesWorkOrder> => {
      const response = await factoriesImportFactoryIntakeItem(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId: input.intakeId },
          body: { itemId: input.itemId },
        }),
      );
      if (!response.data?.order) {
        throw new Error("SuperPlane could not import the item.");
      }
      return response.data.order;
    },
    onSuccess: (order) => {
      applyWorkOrderToListCaches(queryClient, organizationId, factoryId, order.id ?? "", order);
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
      if (order.id) {
        queryClient.setQueryData(factoryQueryKeys.workOrderDetail(organizationId, factoryId, order.id), order);
        void queryClient.invalidateQueries({
          queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, order.id),
        });
      }
    },
  });
}

export type RefreshBacklogResult = {
  archivedCount: number;
  failedItemCount: number;
  failedSourceCount: number;
};

export function useRefreshBacklog(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (): Promise<RefreshBacklogResult> => {
      const response = await factoriesRefreshBacklog(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {},
        }),
      );
      return {
        archivedCount: response.data?.archivedCount ?? 0,
        failedItemCount: response.data?.failedItemCount ?? 0,
        failedSourceCount: response.data?.failedSourceCount ?? 0,
      };
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
    },
  });
}
