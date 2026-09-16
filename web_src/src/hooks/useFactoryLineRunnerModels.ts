import { useQuery } from "@tanstack/react-query";

import { factoriesListFactoryLineRunnerModels } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function factoryLineRunnerModelsQueryKey(organizationId: string, factoryId: string, lineName: string) {
  return ["factories", organizationId, factoryId, "line-runner-models", lineName] as const;
}

export function useFactoryLineRunnerModels(
  organizationId: string | undefined,
  factoryId: string | undefined,
  lineName: string | undefined,
  enabled = true,
) {
  const org = organizationId ?? "";
  const factory = factoryId ?? "";
  const line = lineName?.trim() ?? "";

  return useQuery({
    queryKey: factoryLineRunnerModelsQueryKey(org, factory, line),
    queryFn: async () => {
      const response = await factoriesListFactoryLineRunnerModels(
        withOrganizationHeader({
          organizationId: org,
          path: { factoryId: factory },
          query: { lineName: line },
        }),
      );
      return response.data?.models ?? [];
    },
    enabled: Boolean(org && factory && line && enabled),
    staleTime: 30 * 1000,
  });
}
