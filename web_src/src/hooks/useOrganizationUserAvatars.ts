import type { SuperplaneUsersUser } from "@/api-client";
import { useQuery } from "@tanstack/react-query";
import { usersListUsers } from "@/api-client/sdk.gen";
import { useMemo } from "react";
import { displayNameInitials, type UserDisplayProfile } from "@/lib/userRefDisplay";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { organizationKeys } from "./useOrganizationData";

function toUserDisplayProfile(user: SuperplaneUsersUser): UserDisplayProfile | null {
  const id = user.metadata?.id?.trim();
  if (!id) {
    return null;
  }

  const name = user.spec?.displayName?.trim() || user.metadata?.email?.trim() || "Unknown User";

  return {
    name,
    initials: displayNameInitials(name),
    avatarUrl: user.status?.accountProviders?.[0]?.avatarUrl,
  };
}

export function useOrganizationUserAvatars(organizationId?: string) {
  const { data: users = [] } = useQuery({
    queryKey: organizationKeys.users(organizationId ?? ""),
    queryFn: async () => {
      const response = await usersListUsers(
        withOrganizationHeader({
          query: {
            domainType: "DOMAIN_TYPE_ORGANIZATION",
            domainId: organizationId!,
            includeRoles: false,
          },
        }),
      );
      return response.data?.users || [];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: !!organizationId,
  });

  return useMemo(() => {
    const map = new Map<string, UserDisplayProfile>();
    if (!organizationId) {
      return map;
    }

    for (const user of users) {
      const profile = toUserDisplayProfile(user);
      if (!profile) {
        continue;
      }

      const id = user.metadata?.id;
      if (id) {
        map.set(id, profile);
      }
    }

    return map;
  }, [organizationId, users]);
}
