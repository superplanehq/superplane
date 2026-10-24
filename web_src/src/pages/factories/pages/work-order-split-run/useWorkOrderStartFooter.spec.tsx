import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { useWorkOrderStartFooter } from "./useWorkOrderStartFooter";
import type { SplitRunFooter } from "./splitRunFooter";
import { START } from "./splitRunFooterActions";

let billing: OrganizationsDescribeOrganizationBillingResponse;
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));
vi.mock("@/hooks/useOrganizationBilling", () => ({ useOrganizationBilling: () => ({ data: billing }) }));
vi.mock("@/hooks/useFactoryUsage", () => ({ useFactoryUsage: () => ({}) }));
vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({ useFactoryLineRunnerModels: () => ({}) }));

const footer: SplitRunFooter = {
  kind: "draft",
  sentence: "Ready to start",
  actions: [START],
};

describe("useWorkOrderStartFooter", () => {
  for (const hasPaymentMethod of [false, true]) {
    it(`blocks paused subscriptions with resume guidance when a saved card is ${hasPaymentMethod ? "present" : "missing"}`, () => {
      billing = {
        hostedExecutionAllowed: false,
        polarSubscriptionStatus: "paused",
        requirePaymentMethod: true,
        hasPaymentMethod,
      };
      const { result } = renderHook(() => useWorkOrderStartFooter(footer, "org-1", "factory-1"));
      expect(result.current.actions[0].disabled).toBe(true);
      expect(result.current.note).toEqual({
        headline: "Subscription paused",
        text: "This agent run is blocked. Resume the subscription in Billing to run agents.",
        cta: { label: "View billing", destination: "billing" },
      });
    });
  }
});
