import { useQuery } from "@tanstack/react-query";
import { factoriesListFactoryAutomations } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useFactorySkillUsage(organizationId: string, factoryId: string, command: string) {
  return useQuery({
    queryKey: ["factory-skill-usage", organizationId, factoryId, command],
    enabled: Boolean(organizationId && factoryId && command),
    retry: false,
    queryFn: async () => {
      const response = await factoriesListFactoryAutomations(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { skillName: command },
        }),
      );
      if (!response.data?.automations) throw new Error("Could not check automation usage");
      return response.data.automations;
    },
  });
}
