import { useTaskModelCatalogContext, type TaskModelCatalogContext } from "./useTaskModelCatalogContext";
import { useQuery } from "@tanstack/react-query";

import { factoriesListFactoryLineRunnerModels, type FactoriesFactoryLlmModel } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export interface FactoryLineRunnerModels {
  models: FactoriesFactoryLlmModel[];
  usesHostedFunding?: boolean;
}

export function factoryLineRunnerModelsQueryKey(
  organizationId: string,
  factoryId: string,
  lineName: string,
  context: TaskModelCatalogContext = {},
) {
  return [
    "factories",
    organizationId,
    factoryId,
    "line-runner-models",
    lineName,
    context.orderId ?? "",
    context.ownerId ?? "",
    context.memberId ?? "",
  ] as const;
}

export function useFactoryLineRunnerModels(
  organizationId: string | undefined,
  factoryId: string | undefined,
  lineName: string | undefined,
  enabled = true,
  orderId?: string,
) {
  const context = useTaskModelCatalogContext(organizationId, factoryId, orderId);
  const org = organizationId ?? "";
  const factory = factoryId ?? "";
  const line = lineName?.trim() ?? "";

  return useQuery({
    queryKey: factoryLineRunnerModelsQueryKey(org, factory, line, context),
    queryFn: async (): Promise<FactoryLineRunnerModels> => {
      const response = await factoriesListFactoryLineRunnerModels(
        withOrganizationHeader({
          organizationId: org,
          path: { factoryId: factory },
          query: { lineName: line, ...(orderId ? { orderId } : {}) },
        }),
      );
      return { models: response.data?.models ?? [], usesHostedFunding: response.data?.usesHostedFunding };
    },
    enabled: Boolean(org && factory && line && enabled),
    staleTime: 30 * 1000,
  });
}
