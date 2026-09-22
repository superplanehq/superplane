import { createContext } from "react";
import { type AuthorizationPermission } from "@/api-client";

export interface PermissionsContextType {
  permissions: AuthorizationPermission[];
  isLoading: boolean;
  canAct: (resource: string, action: string) => boolean;
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
  currentUserId: undefined,
  browserNotificationPreferences: {
    enabled: false,
    showWhileViewing: true,
  },
});
