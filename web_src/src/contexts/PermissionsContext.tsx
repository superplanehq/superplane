import React, { createContext, useCallback, useContext, useMemo } from "react";
import { type AuthorizationPermission } from "@/api-client";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useMe } from "@/hooks/useMe";

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
  const organizationId = useOrganizationId();
  const { data: me, isLoading: meLoading } = useMe();

  const userId = me?.id;
  const permissions = me?.permissions ?? [];

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

  const isLoading = !organizationId || meLoading || (!!organizationId && !userId);

  return (
    <PermissionsContext.Provider value={{ permissions, isLoading, canAct }}>{children}</PermissionsContext.Provider>
  );
};
