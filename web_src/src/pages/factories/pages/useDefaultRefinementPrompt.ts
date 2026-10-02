import { factoriesMaterializeFactoryAutomationDefaults } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";

import { refinementPromptFromAutomationDefaults } from "../lib/refinementPromptFromAutomationDefaults";

export function useDefaultRefinementPrompt(input: {
  organizationId: string;
  factoryId: string;
  automationId?: string;
}) {
  const automationId = input.automationId ?? "";
  const query = useQuery({
    queryKey: ["factory-automation-defaults", input.organizationId, input.factoryId, automationId, "refinement-prompt"],
    enabled: Boolean(input.organizationId && input.factoryId && automationId),
    retry: false,
    queryFn: async () => {
      const response = await factoriesMaterializeFactoryAutomationDefaults(
        withOrganizationHeader({
          organizationId: input.organizationId,
          path: { factoryId: input.factoryId, automationId },
          body: {},
        }),
      );
      return refinementPromptFromAutomationDefaults(response.data?.canvasYaml ?? "");
    },
  });

  return {
    prompt: query.data || undefined,
    failed: query.isError,
    retry: () => {
      void query.refetch();
    },
  };
}
