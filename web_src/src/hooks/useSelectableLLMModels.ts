import { useQuery } from "@tanstack/react-query";

import { organizationsListSelectableLlmModels } from "@/api-client";
import {
  selectableLLMModelsFromResponse,
  type SelectableLLMModel,
  type SelectableLLMSourceID,
} from "@/lib/selectableLLMModels";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function selectableLLMModelsQueryKey(organizationId: string, factoryId?: string, sources?: string) {
  return ["organizations", organizationId, "selectable-llm-models", factoryId ?? "", sources ?? ""] as const;
}

export function useSelectableLLMModels(
  organizationId: string | undefined,
  options?: { factoryId?: string; sources?: SelectableLLMSourceID[]; enabled?: boolean },
) {
  const factoryId = options?.factoryId;
  const sources = options?.sources;
  const enabled = options?.enabled ?? true;
  return useQuery({
    queryKey: selectableLLMModelsQueryKey(organizationId ?? "", factoryId, sources?.join(",") ?? ""),
    queryFn: async (): Promise<SelectableLLMModel[]> => {
      const response = await organizationsListSelectableLlmModels(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
          query: { factoryId },
        }),
      );
      return selectableLLMModelsFromResponse(response.data?.models ?? [], sources);
    },
    enabled: Boolean(organizationId && enabled),
    staleTime: 30 * 1000,
  });
}
