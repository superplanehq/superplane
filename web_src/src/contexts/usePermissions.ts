import { useContext } from "react";

import { PermissionsContext } from "./permissionsContextState";

export function usePermissions() {
  const context = useContext(PermissionsContext);
  if (!context) {
    throw new Error("usePermissions must be used within a PermissionsProvider");
  }
  return context;
}
