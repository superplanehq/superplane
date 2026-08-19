import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap, createOrgUserDisplayLookup, type OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { useMemo } from "react";

export function useOrgUserLookup(organizationId: string): {
  resolveUser: OrgUserDisplayLookup;
  isLoading: boolean;
} {
  const { data: users = [], isLoading } = useOrganizationUsers(organizationId);

  const resolveUser = useMemo(() => {
    const usersById = buildOrgUserDisplayMap(users);
    return createOrgUserDisplayLookup(usersById);
  }, [users]);

  return { resolveUser, isLoading };
}
