import { useQuery } from "@tanstack/react-query";

import { parseAccountOrganizations, type AccountOrganization } from "@/lib/accountOrganizations";

export type { AccountOrganization };

export const accountOrganizationsQueryKey = ["account-organizations"] as const;

async function fetchAccountOrganizations(): Promise<AccountOrganization[]> {
  const response = await fetch("/organizations", { credentials: "include" });
  if (!response.ok) {
    throw new Error("Failed to load organizations");
  }
  return parseAccountOrganizations(await response.json());
}

export function useAccountOrganizations() {
  return useQuery({
    queryKey: accountOrganizationsQueryKey,
    queryFn: fetchAccountOrganizations,
    staleTime: 5 * 60 * 1000,
  });
}
