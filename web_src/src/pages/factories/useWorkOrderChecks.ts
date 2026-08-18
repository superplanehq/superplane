import { useQuery } from "@tanstack/react-query";

import type { WorkOrderCheckPresentation } from "./WorkOrderChecksSection";

/**
 * Checks reported by automations for one work order.
 *
 * Temporary mock-phase hook: it fetches the future
 * `GET /orders/{orderId}/checks` endpoint directly so the Storybook fixture
 * can serve it. The live API does not expose it yet, so any failure resolves
 * to an empty list and the section stays hidden. Replace with the generated
 * SDK call once the checks proto lands.
 */
export function useWorkOrderChecks(organizationId: string, factoryId: string, orderId: string) {
  return useQuery({
    queryKey: ["factories", organizationId, factoryId, "workOrderChecks", orderId],
    queryFn: async (): Promise<WorkOrderCheckPresentation[]> => {
      try {
        const response = await fetch(`/api/v1/factories/${factoryId}/orders/${orderId}/checks`, {
          headers: { "x-organization-id": organizationId },
        });
        if (!response.ok) {
          return [];
        }
        const payload = (await response.json()) as { checks?: WorkOrderCheckPresentation[] };
        return payload.checks ?? [];
      } catch {
        return [];
      }
    },
    enabled: Boolean(organizationId && factoryId && orderId),
  });
}
