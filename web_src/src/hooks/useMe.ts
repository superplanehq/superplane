import { useQuery } from "@tanstack/react-query";
import { meMe } from "@/api-client";
import { useOrganizationId, withOrganizationHeader } from "@/utils/withOrganizationHeader";

export const meKeys = {
  me: ["me"] as const,
};

export const useMe = () => {
  const organizationId = useOrganizationId();
  return useQuery({
    queryKey: meKeys.me,
    queryFn: async () => {
      const response = await meMe(withOrganizationHeader());
      return response.data ?? null;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: !!organizationId,
  });
};
