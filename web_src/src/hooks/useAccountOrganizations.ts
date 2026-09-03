import { useQuery } from "@tanstack/react-query";

export interface AccountOrganization {
  id: string;
  slug: string;
  name: string;
}

export const accountOrganizationsQueryKey = ["account-organizations"] as const;

async function fetchAccountOrganizations(): Promise<AccountOrganization[]> {
  const response = await fetch("/organizations", { credentials: "include" });
  if (!response.ok) {
    throw new Error("Failed to load organizations");
  }
  const body: unknown = await response.json();
  if (!Array.isArray(body)) {
    return [];
  }
  return body.filter((entry): entry is AccountOrganization => {
    return Boolean(
      entry &&
        typeof entry === "object" &&
        typeof (entry as AccountOrganization).id === "string" &&
        typeof (entry as AccountOrganization).slug === "string" &&
        typeof (entry as AccountOrganization).name === "string",
    );
  });
}

export function useAccountOrganizations() {
  return useQuery({
    queryKey: accountOrganizationsQueryKey,
    queryFn: fetchAccountOrganizations,
    staleTime: 5 * 60 * 1000,
  });
}
