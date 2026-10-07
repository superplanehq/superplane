import { useQuery } from "@tanstack/react-query";
import { meMe } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useOrganizationId } from "@/hooks/useOrganizationId";

export const meKeys = {
  all: ["me"] as const,
  organization: (organizationId: string) => [...meKeys.all, organizationId] as const,
  me: (organizationId: string, includePermissions: boolean = true) =>
    [...meKeys.organization(organizationId), includePermissions] as const,
};

export const useMe = (includePermissions: boolean = true, organizationIdOverride?: string | null) => {
  const organizationIdFromRoute = useOrganizationId();
  const organizationId = organizationIdOverride !== undefined ? organizationIdOverride : organizationIdFromRoute;

  return useQuery({
    queryKey: organizationId ? meKeys.me(organizationId, includePermissions) : ["me", "unknown"],
    queryFn: async () => {
      const response = await meMe(withOrganizationHeader({ organizationId, query: { includePermissions } }));
      return response.data?.user ?? null;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    refetchOnWindowFocus: "always",
    enabled: !!organizationId,
  });
};
