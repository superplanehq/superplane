import { useQuery } from "@tanstack/react-query";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type RunnerFleet = {
  id: string;
  provisioner?: string;
  arch?: string;
  size?: string;
  created_at_unix?: number;
};

export type RunnerFleetsResponse = {
  configured: boolean;
  fleets: RunnerFleet[];
};

export const runnerFleetKeys = {
  all: ["runner-fleets"] as const,
  list: (organizationId?: string) => [...runnerFleetKeys.all, organizationId ?? "current"] as const,
};

export function useRunnerFleets(organizationId?: string) {
  return useQuery({
    queryKey: runnerFleetKeys.list(organizationId),
    queryFn: async (): Promise<RunnerFleetsResponse> => {
      const response = await fetch(
        "/api/v1/runner/fleets",
        withOrganizationHeader({
          organizationId,
          credentials: "include",
        }),
      );

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text.trim() || "Failed to load runner fleets");
      }

      const data = (await response.json()) as Partial<RunnerFleetsResponse>;
      return {
        configured: data.configured ?? false,
        fleets: data.fleets ?? [],
      };
    },
    staleTime: 60 * 1000,
  });
}
