import React, { useCallback, useMemo } from "react";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useMe } from "@/hooks/useMe";
import { useIsSupportSession } from "@/hooks/useSupportSession";

import { PermissionsContext } from "./permissionsContextState";

interface PermissionsProviderProps {
  children: React.ReactNode;
  organizationId?: string;
}

export function PermissionsProvider({ children, organizationId: organizationIdOverride }: PermissionsProviderProps) {
  const routeOrganizationId = useOrganizationId();
  const organizationId = organizationIdOverride ?? routeOrganizationId;
  const { data: me, isLoading: meLoading } = useMe(true, organizationId);
  const inSupportSession = useIsSupportSession();

  const permissions = useMemo(() => me?.permissions ?? [], [me?.permissions]);
  const browserNotificationPreferences = useMemo(
    () => ({
      enabled: me?.browserNotificationPreferences?.enabled ?? false,
      showWhileViewing: me?.browserNotificationPreferences?.showWhileViewing ?? true,
    }),
    [me?.browserNotificationPreferences?.enabled, me?.browserNotificationPreferences?.showWhileViewing],
  );

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

  const hasPermission = useCallback(
    (resource: string, action: string) => {
      if (!resource || !action) return false;
      return permissionSet.has(`${resource.toLowerCase()}:${action.toLowerCase()}`);
    },
    [permissionSet],
  );

  const canAct = useCallback(
    (resource: string, action: string) => {
      if (inSupportSession && action.toLowerCase() !== "read") return false;
      return hasPermission(resource, action);
    },
    [hasPermission, inSupportSession],
  );

  const isLoading = !organizationId || meLoading;

  return (
    <PermissionsContext.Provider
      value={{
        permissions,
        isLoading,
        canAct,
        hasPermission,
        currentUserId: me?.id,
        browserNotificationPreferences,
      }}
    >
      {children}
    </PermissionsContext.Provider>
  );
}
