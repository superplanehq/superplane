import { createContext } from "react";
import { type AuthorizationPermission } from "@/api-client";

export interface PermissionsContextType {
  permissions: AuthorizationPermission[];
  isLoading: boolean;
  /** False for every action except "read" during a read-only support session. */
  canAct: (resource: string, action: string) => boolean;
  /** The user's real permission. Use for page access, not for actions. */
  hasPermission: (resource: string, action: string) => boolean;
  currentUserId?: string;
  browserNotificationPreferences: {
    enabled: boolean;
    showWhileViewing: boolean;
  };
}

export const PermissionsContext = createContext<PermissionsContextType>({
  permissions: [],
  isLoading: false,
  canAct: () => false,
  hasPermission: () => false,
  currentUserId: undefined,
  browserNotificationPreferences: {
    enabled: false,
    showWhileViewing: true,
  },
});
