import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";

export type RunnerFleetOption = {
  id: string;
  name: string;
};

async function fetchRunnerFleets(organizationId: string): Promise<RunnerFleetOption[]> {
  const res = await fetch("/api/v1/runner-fleets", {
    credentials: "include",
    ...withOrganizationHeader({ organizationId }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text.trim() || "Failed to load machine types");
  }
  const data = (await res.json()) as RunnerFleetOption[];
  return Array.isArray(data) ? data : [];
}

export function useRunnerFleets(organizationId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ["runner-fleets", organizationId],
    queryFn: () => fetchRunnerFleets(organizationId!),
    enabled: enabled && !!organizationId,
    staleTime: 60_000,
  });
}
