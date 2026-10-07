import { useQuery } from "@tanstack/react-query";

import { organizationsListHostedLlmModels } from "@/api-client";
import type { OrganizationsListHostedLlmModelsResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function hostedLLMModelsQueryKey(organizationId: string, provider: string, factoryId?: string) {
  return ["organizations", organizationId, "hosted-llm-models", provider, factoryId ?? ""] as const;
}

export function useHostedLLMModels(
  organizationId: string | undefined,
  provider: string | undefined,
  enabled: boolean,
  factoryId?: string,
) {
  return useQuery({
    queryKey: hostedLLMModelsQueryKey(organizationId ?? "", provider ?? "", factoryId),
    queryFn: async (): Promise<OrganizationsListHostedLlmModelsResponse> => {
      const response = await organizationsListHostedLlmModels(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
          query: { provider, factoryId },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && provider && enabled),
    staleTime: 30 * 1000,
  });
}
