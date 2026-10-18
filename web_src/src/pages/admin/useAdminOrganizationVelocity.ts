import { useQuery } from "@tanstack/react-query";

import type { FactoriesDescribeFactoryVelocityResponse } from "@/api-client";

import type { VelocityPeriodDays } from "@/pages/factories/lib/factoryVelocityReport";

export interface AdminVelocityFactory {
  id: string;
  name: string;
}

export type AdminOrganizationVelocity = FactoriesDescribeFactoryVelocityResponse & {
  factories: AdminVelocityFactory[];
  factoryId?: string;
  periodDays?: number;
};

export function useAdminOrganizationVelocity(
  organizationId: string,
  periodDays: VelocityPeriodDays,
  factoryId?: string,
) {
  return useQuery({
    queryKey: ["admin", "organizations", organizationId, "velocity", periodDays, factoryId ?? ""] as const,
    queryFn: () => fetchAdminOrganizationVelocity(organizationId, periodDays, factoryId),
    enabled: Boolean(organizationId),
  });
}

async function fetchAdminOrganizationVelocity(
  organizationId: string,
  periodDays: VelocityPeriodDays,
  factoryId?: string,
): Promise<AdminOrganizationVelocity> {
  const params = new URLSearchParams({ period_days: String(periodDays) });
  if (factoryId) {
    params.set("factory_id", factoryId);
  }

  const response = await fetch(`/admin/api/organizations/${organizationId}/velocity?${params}`, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Could not load velocity.");
  }
  return (await response.json()) as AdminOrganizationVelocity;
}
