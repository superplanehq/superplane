import { useQuery } from "@tanstack/react-query";

import { organizationsListHostedLlmModels } from "@/api-client";
import type { OrganizationsListHostedLlmModelsResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useHostedLLMModels(organizationId: string | undefined, provider: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["organizations", organizationId, "hosted-llm-models", provider] as const,
    queryFn: async (): Promise<OrganizationsListHostedLlmModelsResponse> => {
      const response = await organizationsListHostedLlmModels(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
          query: { provider },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && provider && enabled),
    staleTime: 30 * 1000,
  });
}
