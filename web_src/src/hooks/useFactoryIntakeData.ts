import {
  factoriesCreateFactoryIntake,
  factoriesListFactoryIntakeRuns,
  factoriesListFactoryIntakes,
  factoriesUpdateFactoryIntake,
} from "@/api-client";
import type {
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeRun,
  FactoriesFactoryIntakeSource,
  FactoryIntakeSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryAppsKey } from "./useFactoryData";

const factoryIntakeQueryKeys = {
  list: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId, "intakes"] as const,
  runs: (organizationId: string, factoryId: string, intakeId: string) =>
    ["factories", organizationId, factoryId, "intakes", intakeId, "runs"] as const,
};

export function factoryIntakesKey(organizationId: string, factoryId: string) {
  return factoryIntakeQueryKeys.list(organizationId, factoryId);
}

export function useFactoryIntakes(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryIntakesKey(organizationId, factoryId),
    queryFn: async (): Promise<FactoriesFactoryIntake[]> => {
      const response = await factoriesListFactoryIntakes(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      return response.data?.intakes ?? [];
    },
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
