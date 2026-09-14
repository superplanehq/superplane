import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { ADMIN_LOCAL_PLAN_COPY, ADMIN_POLAR_MANAGED_PLAN_COPY, OrgLLMCreditSection } from "./OrgLLMCreditSection";

const credit = {
  remaining_credit_cents: 5000,
  grant_total_cents: 5000,
  superplane_grant_cents: 5000,
  purchased_credit_cents: 0,
  hosted_billed_cents: 0,
  markup_bps: 2000,
  markup_override_bps: null,
  warning: false,
};

function billingPlan(polarManaged: boolean) {
  return {
    plan: polarManaged ? "business" : "trial",
    plan_source: polarManaged ? "polar" : "system",
    polar_subscription_status: polarManaged ? "active" : "",
    polar_managed: polarManaged,
    trial_ends_at: null,
    current_period_end: null,
  };
}

function mockOrgBillingFetch(polarManaged: boolean) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/llm-credit") && !url.includes("/grants")) {
        return new Response(JSON.stringify(credit), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      if (url.includes("/billing-plan")) {
        return new Response(JSON.stringify(billingPlan(polarManaged)), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      return new Response("not found", { status: 404 });
    }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("OrgLLMCreditSection billing plan", () => {
  it("keeps Save plan when Polar is not set up", async () => {
    mockOrgBillingFetch(false);
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeEnabled();
    expect(screen.getByTestId("admin-org-billing-plan-save")).toBeInTheDocument();
    expect(screen.getByText(ADMIN_LOCAL_PLAN_COPY)).toBeInTheDocument();
  });

  it("disables the plan control when Polar manages billing", async () => {
    mockOrgBillingFetch(true);
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
    expect(screen.getByText(ADMIN_POLAR_MANAGED_PLAN_COPY)).toBeInTheDocument();
  });
});
