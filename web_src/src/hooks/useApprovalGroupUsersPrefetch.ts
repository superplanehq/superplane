import { useMemo } from "react";
import { useQueries } from "@tanstack/react-query";
import { groupsListGroupUsers } from "@/api-client";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

export const useApprovalGroupUsersPrefetch = ({
  organizationId,
  groupNames,
  enabled = true,
}: {
  organizationId?: string;
  groupNames: string[];
  enabled?: boolean;
}) => {
  const organizationIdValue = organizationId || "";
  const uniqueGroupNames = useMemo(() => {
    return Array.from(new Set(groupNames)).filter((name) => !!name);
  }, [groupNames]);

  const groupUsersQueries = useQueries({
    queries: uniqueGroupNames.map((groupName) => ({
      queryKey: organizationKeys.groupUsers(organizationIdValue, groupName),
      queryFn: async () => {
        const response = await groupsListGroupUsers(
          withOrganizationHeader({
            path: { groupName },
            query: { domainId: organizationIdValue, domainType: "DOMAIN_TYPE_ORGANIZATION" },
          }),
        );
        return response.data?.users || [];
      },
      staleTime: 5 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
      enabled: enabled && !!organizationIdValue && !!groupName,
    })),
  });

  const updatedAt = useMemo(() => {
    return groupUsersQueries.map((query) => query.dataUpdatedAt).join("|");
  }, [groupUsersQueries]);

  return { updatedAt };
};
