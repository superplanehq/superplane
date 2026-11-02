import { useQuery } from "@tanstack/react-query";

import { meListCodexUsage } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useCodexUsage(organizationId: string, enabled = true) {
  const { currentUserId } = usePermissions();
  return useQuery({
    queryKey: ["me", organizationId, "codex-usage", currentUserId],
    queryFn: async () => {
      const response = await meListCodexUsage(withOrganizationHeader({ organizationId }));
      if (!response.data) throw new Error("Could not load Codex usage.");
      return response.data.connections ?? [];
    },
    enabled: enabled && Boolean(organizationId),
    refetchInterval: false,
    refetchOnWindowFocus: false,
  });
}
