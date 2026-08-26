import {
  factoriesCreateFactoryIntake,
  factoriesImportFactoryIntakeItem,
  factoriesListFactoryIntakeRuns,
  factoriesListFactoryIntakes,
  factoriesSearchFactoryIntakeItems,
  factoriesUpdateFactoryIntake,
} from "@/api-client";
import type {
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeItem,
  FactoriesFactoryIntakeRun,
  FactoriesFactoryIntakeSource,
  FactoriesWorkOrder,
  FactoryIntakeSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryAppsKey, factoryQueryKeys } from "./useFactoryData";

const factoryIntakeQueryKeys = {
  list: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId, "intakes"] as const,
  runs: (organizationId: string, factoryId: string, intakeId: string) =>
    ["factories", organizationId, factoryId, "intakes", intakeId, "runs"] as const,
  items: (organizationId: string, factoryId: string, intakeId: string, query: string, limit: number) =>
    ["factories", organizationId, factoryId, "intakes", intakeId, "items", query, limit] as const,
};

export function factoryIntakesKey(organizationId: string, factoryId: string) {
  return factoryIntakeQueryKeys.list(organizationId, factoryId);
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
    queryKey: factoryIntakeQueryKeys.runs(organizationId, factoryId, intakeId ?? ""),
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
    // Items leave the analysis on their own. The open list has to follow them,
    // because a new intake starts with a batch that drains within minutes.
    refetchInterval: 10_000,
  });
}

export function useCreateFactoryIntake(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { source: FactoriesFactoryIntakeSource; name?: string; confidencePct?: number }) => {
      const response = await factoriesCreateFactoryIntake(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            source: input.source,
            name: input.name,
            confidencePct: input.confidencePct,
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
    },
  });
}

export function useUpdateFactoryIntake(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { intakeId: string; name?: string; settings?: FactoryIntakeSettings }) => {
      const response = await factoriesUpdateFactoryIntake(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, intakeId: input.intakeId },
          body: {
            name: input.name,
            settings: input.settings,
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
      queryClient.setQueryData<FactoriesWorkOrder[]>(
        factoryQueryKeys.workOrders(organizationId, factoryId),
        (current) => upsertImportedWorkOrder(current, order),
      );
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      if (order.id) {
        queryClient.setQueryData(factoryQueryKeys.workOrderDetail(organizationId, factoryId, order.id), order);
        void queryClient.invalidateQueries({
          queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, order.id),
        });
      }
    },
  });
}

function upsertImportedWorkOrder(
  current: FactoriesWorkOrder[] | undefined,
  order: FactoriesWorkOrder,
): FactoriesWorkOrder[] {
  if (!order.id) {
    return current ?? [];
  }
  if (!current) {
    return [order];
  }
  if (current.some((existing) => existing.id === order.id)) {
    return current.map((existing) => (existing.id === order.id ? order : existing));
  }
  return [order, ...current];
}
