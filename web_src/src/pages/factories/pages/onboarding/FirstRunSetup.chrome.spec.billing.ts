import { vi } from "bun:test";
import type { OrganizationsBillingPlanOption } from "@/api-client";

export const defaultBillingState = {
  enabled: false,
  byok: false,
  planActive: false,
  hasPaymentMethod: false,
};
export const billingState: typeof defaultBillingState & { availablePlans?: OrganizationsBillingPlanOption[] } = {
  ...defaultBillingState,
};
const billingData = () => ({
  billingEnabled: billingState.enabled,
  availablePlans: billingState.availablePlans,
  productFeatures: { byo_key: true },
  planCategory: billingState.planActive ? "paid" : "none",
  creditPurchaseAllowed: billingState.planActive,
  hasPaymentMethod: billingState.hasPaymentMethod,
});
export const refetchBilling = vi.fn(async () => ({ data: billingData() }));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({ data: billingData(), isPending: false, isSuccess: true, refetch: refetchBilling }),
}));
