import React, { createContext, useCallback, useContext, useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { AuthorizationPermission, meMe, usersListUserPermissions } from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

interface PermissionsContextType {
  permissions: AuthorizationPermission[];
  isLoading: boolean;
  canAct: (resource: string, action: string) => boolean;
}

const PermissionsContext = createContext<PermissionsContextType>({
  permissions: [],
  isLoading: false,
  canAct: () => false,
});

export const usePermissions = () => {
  const context = useContext(PermissionsContext);
  if (!context) {
    throw new Error("usePermissions must be used within a PermissionsProvider");
  }
  return context;
};

interface PermissionsProviderProps {
  children: React.ReactNode;
}

export const PermissionsProvider: React.FC<PermissionsProviderProps> = ({ children }) => {
  const { organizationId } = useParams<{ organizationId?: string }>();

  const meQuery = useQuery({
    queryKey: ["me", organizationId],
    queryFn: async () => {
      const response = await meMe(withOrganizationHeader());
      return response.data || null;
    },
    enabled: !!organizationId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const userId = meQuery.data?.id;

  const permissionsQuery = useQuery({
    queryKey: ["permissions", organizationId, userId],
    queryFn: async () => {
      const response = await usersListUserPermissions(
        withOrganizationHeader({
          path: { userId: userId! },
          query: { domainType: "DOMAIN_TYPE_ORGANIZATION", domainId: organizationId },
        }),
      );
      return response.data?.permissions || [];
    },
    enabled: !!organizationId && !!userId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const permissions = permissionsQuery.data ?? [];

  const permissionSet = useMemo(() => {
    return new Set(
      permissions
        .map((perm) => {
          const resource = perm.resource?.toLowerCase();
          const action = perm.action?.toLowerCase();
          if (!resource || !action) return null;
          return `${resource}:${action}`;
        })
        .filter((value): value is string => !!value),
    );
  }, [permissions]);

  const canAct = useCallback(
    (resource: string, action: string) => {
      if (!resource || !action) return false;
      return permissionSet.has(`${resource.toLowerCase()}:${action.toLowerCase()}`);
    },
    [permissionSet],
  );

  const isLoading = meQuery.isLoading || permissionsQuery.isLoading;

  return (
    <PermissionsContext.Provider value={{ permissions, isLoading, canAct }}>{children}</PermissionsContext.Provider>
  );
};
