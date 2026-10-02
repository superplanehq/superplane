import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  ADMIN_LOCAL_PLAN_COPY,
  ADMIN_PLAN_UNKNOWN_COPY,
  ADMIN_POLAR_MANAGED_PLAN_COPY,
  ADMIN_TRIAL_ENDS_ERROR,
  ADMIN_TRIAL_ENDS_HELP,
  ADMIN_TRIAL_ENDS_LABEL,
  OrgLLMCreditSection,
} from "./OrgLLMCreditSection";
import { prefilledTrialEndDate } from "./useOrgLLMCredit";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

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

function futureUtcDate(daysAhead: number): string {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + daysAhead))
    .toISOString()
    .slice(0, 10);
}

function billingPlan(
  polarManaged: boolean,
  trialEndsAt: string | null = null,
  plan = polarManaged ? "business" : "trial",
) {
  return {
    plan,
    plan_source: polarManaged ? "polar" : "system",
    polar_subscription_status: polarManaged ? "active" : "",
    polar_managed: polarManaged,
    trial_ends_at: trialEndsAt,
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

  it("disables the plan control when billing plan is unknown", async () => {
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
          return new Response("failed", { status: 500 });
        }
        return new Response("not found", { status: 404 });
      }),
    );
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
    expect(screen.getByText(ADMIN_PLAN_UNKNOWN_COPY)).toBeInTheDocument();
  });

  it("shows the trial end and date field only for Trial", async () => {
    const trialDate = futureUtcDate(40);
    const trialEndsAt = `${trialDate}T23:59:59Z`;
    mockOrgBillingFetch(false);
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
          return new Response(JSON.stringify(billingPlan(false, trialEndsAt, "trial")), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("not found", { status: 404 });
      }),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-trial-ends-input")).toHaveValue(trialDate);
    expect(screen.getByText(ADMIN_TRIAL_ENDS_LABEL)).toBeInTheDocument();
    expect(screen.getByText(ADMIN_TRIAL_ENDS_HELP)).toBeInTheDocument();

    await user.click(screen.getByTestId("admin-org-billing-plan"));
    await user.click(await screen.findByRole("option", { name: "Business" }));

    expect(screen.queryByTestId("admin-org-trial-ends-input")).not.toBeInTheDocument();
    expect(screen.getByTestId("admin-org-trial-ends")).toHaveTextContent(trialDate);
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_HELP)).not.toBeInTheDocument();
  });

  it("hides the trial end when Polar manages billing", async () => {
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
          return new Response(JSON.stringify(billingPlan(true, "2026-12-20T23:59:59Z", "business")), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("not found", { status: 404 });
      }),
    );
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-trial-ends-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-trial-ends")).not.toBeInTheDocument();
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_LABEL)).not.toBeInTheDocument();
  });

  it("prefills 14 days from today when no future trial end exists", async () => {
    mockOrgBillingFetch(false);
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-trial-ends-input")).toHaveValue(prefilledTrialEndDate(null));
  });

  it("sends the chosen UTC day end and rejects a past date", async () => {
    const futureDate = futureUtcDate(30);
    const requests: Array<{ url: string; body: string | null }> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method ?? "GET";
        if (method === "PUT") {
          requests.push({ url, body: typeof init?.body === "string" ? init.body : null });
          return new Response(JSON.stringify(billingPlan(false, `${futureDate}T23:59:59Z`, "trial")), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (url.includes("/llm-credit") && !url.includes("/grants")) {
          return new Response(JSON.stringify(credit), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (url.includes("/billing-plan")) {
          return new Response(JSON.stringify(billingPlan(false, `${futureUtcDate(40)}T23:59:59Z`, "trial")), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("not found", { status: 404 });
      }),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);

    const input = await screen.findByTestId("admin-org-trial-ends-input");
    fireEvent.change(input, { target: { value: "2020-01-01" } });
    expect(screen.getByText(ADMIN_TRIAL_ENDS_ERROR)).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));
    expect(requests).toHaveLength(0);

    fireEvent.change(input, { target: { value: futureDate } });
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_ERROR)).not.toBeInTheDocument();
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));
    await waitFor(() => expect(requests).toHaveLength(1));
    expect(requests[0]?.url).toContain("/billing-plan");
    expect(JSON.parse(requests[0]?.body ?? "{}")).toEqual({
      plan: "trial",
      trial_ends_at: `${futureDate}T23:59:59Z`,
    });
  });
});
