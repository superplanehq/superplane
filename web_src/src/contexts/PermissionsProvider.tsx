import React, { useCallback, useMemo } from "react";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useMe } from "@/hooks/useMe";

import { PermissionsContext } from "./permissionsContextState";

interface PermissionsProviderProps {
  children: React.ReactNode;
}

export function PermissionsProvider({ children }: PermissionsProviderProps) {
  const organizationId = useOrganizationId();
  const { data: me, isLoading: meLoading } = useMe();

  const permissions = useMemo(() => me?.permissions ?? [], [me?.permissions]);

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

  const isLoading = !organizationId || meLoading || (!!organizationId && meLoading);

  return (
    <PermissionsContext.Provider value={{ permissions, isLoading, canAct }}>{children}</PermissionsContext.Provider>
  );
}
