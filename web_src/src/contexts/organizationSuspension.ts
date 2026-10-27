import { createContext, useContext } from "react";

export const OrganizationSuspensionContext = createContext(false);

export function useIsOrganizationSuspended() {
  return useContext(OrganizationSuspensionContext);
}
