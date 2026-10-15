import { useContext } from "react";

import { PermissionsContext, type PermissionsContextType } from "./permissionsContextState";

const missingPermissions: PermissionsContextType = {
  permissions: [],
  isLoading: false,
  canAct: () => false,
  currentUserId: undefined,
  browserNotificationPreferences: {
    enabled: false,
    showWhileViewing: true,
  },
};

export function usePermissions() {
  return useContext(PermissionsContext) ?? missingPermissions;
}
