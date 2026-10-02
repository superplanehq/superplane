import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  ADMIN_LOCAL_PLAN_COPY,
  ADMIN_NO_ACTIVE_TRIAL_COPY,
  ADMIN_PLAN_UNKNOWN_COPY,
  ADMIN_POLAR_MANAGED_PLAN_COPY,
  OrgLLMCreditSection,
} from "./OrgLLMCreditSection";
import { CREDIT_BALANCE_CHANGED_COPY } from "./useOrgLLMCredit";
import { showErrorToast } from "@/lib/toast";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

const trialEnd = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();

const credit = {
  remaining_credit_cents: 9500,
  grant_total_cents: 10000,
  superplane_grant_cents: 7000,
  purchased_credit_cents: 3000,
  hosted_billed_cents: 500,
  welcome_remaining_cents: 4500,
  included_remaining_cents: 0,
  purchased_remaining_cents: 3000,
  admin_remaining_cents: 2000,
  welcome_credit_expires_at: trialEnd,
  markup_bps: 2000,
  markup_override_bps: null,
  warning: false,
};

const grants = [
  {
    id: "grant-2",
    kind: "trial_adjustment",
    amountCents: "-500",
    note: "Support correction",
    actorName: "Ada",
    createdAt: new Date().toISOString(),
  },
  { id: "grant-1", kind: "welcome", amountCents: "5000", note: "", createdAt: new Date().toISOString() },
];

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

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function mockOrgCreditFetch(
  options: {
    polarManaged?: boolean;
    planStatus?: number;
    creditData?: typeof credit;
    onPutBalance?: (body: unknown) => Response;
  } = {},
) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/llm-credit/balances") && init?.method === "PUT") {
      const body = JSON.parse(String(init.body));
      return options.onPutBalance ? options.onPutBalance(body) : jsonResponse(options.creditData ?? credit);
    }
    if (url.endsWith("/llm-credit/grants")) {
      return jsonResponse({ grants });
    }
    if (url.endsWith("/llm-credit")) {
      return jsonResponse(options.creditData ?? credit);
    }
    if (url.includes("/billing-plan")) {
      if (options.planStatus) {
        return new Response("failed", { status: options.planStatus });
      }
      return jsonResponse(billingPlan(options.polarManaged ?? false));
    }
    return new Response("not found", { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.mocked(showErrorToast).mockClear();
});

describe("OrgLLMCreditSection billing plan", () => {
  it("keeps Save plan when Polar is not set up", async () => {
    mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeEnabled();
    expect(screen.getByTestId("admin-org-billing-plan-save")).toBeInTheDocument();
    expect(screen.getByText(ADMIN_LOCAL_PLAN_COPY)).toBeInTheDocument();
  });

  it("disables the plan control when Polar manages billing", async () => {
    mockOrgCreditFetch({ polarManaged: true });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
    expect(screen.getByText(ADMIN_POLAR_MANAGED_PLAN_COPY)).toBeInTheDocument();
  });

  it("disables the plan control when billing plan is unknown", async () => {
    mockOrgCreditFetch({ planStatus: 500 });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
    expect(screen.getByText(ADMIN_PLAN_UNKNOWN_COPY)).toBeInTheDocument();
  });
});

describe("OrgLLMCreditSection balances", () => {
  it("shows the remaining balance of each credit type and the credit history", async () => {
    mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-total-remaining")).toHaveTextContent("$95.00");
    expect(screen.getByTestId("admin-org-credit-trial-remaining")).toHaveTextContent("$45.00");
    expect(screen.getByTestId("admin-org-credit-included-remaining")).toHaveTextContent("$0.00");
    expect(screen.getByTestId("admin-org-credit-topup-remaining")).toHaveTextContent("$30.00");
    expect(screen.getByTestId("admin-org-credit-grant-remaining")).toHaveTextContent("$20.00");
    expect(screen.getByTestId("admin-org-credit-trial-target")).toHaveValue("45.00");
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();

    const history = screen.getByTestId("admin-org-credit-history");
    expect(await within(history).findByText("Trial adjustment")).toBeInTheDocument();
    expect(within(history).getByText("-$5.00")).toBeInTheDocument();
    expect(within(history).getByText("Support correction · Adjusted by Ada")).toBeInTheDocument();
  });

  it("sends the target and the displayed balance when an admin sets a balance", async () => {
    const user = userEvent.setup();
    const fetchMock = mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    const trial = await screen.findByTestId("admin-org-credit-trial-target");
    await user.clear(trial);
    await user.type(trial, "10");
    expect(screen.getByTestId("admin-org-credit-trial-change")).toHaveTextContent("Ledger change: -$35.00");
    await user.type(screen.getByTestId("admin-org-credit-note"), "Correct trial");
    await user.click(screen.getByTestId("admin-org-credit-trial-save"));

    await waitFor(() => {
      const put = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT");
      expect(put).toBeDefined();
      expect(String(put?.[0])).toBe("/admin/api/organizations/org-1/llm-credit/balances");
      expect(JSON.parse(String(put?.[1]?.body))).toEqual({
        bucket: "trial",
        target_cents: 1000,
        expected_remaining_cents: 4500,
        note: "Correct trial",
      });
    });
  });

  it("keeps Set disabled when the balance does not change", async () => {
    mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-grant-save")).toBeDisabled();
    expect(screen.getByTestId("admin-org-credit-grant-change")).toHaveTextContent("No change.");
  });

  it("blocks trial changes when the organization has no active trial credit", async () => {
    mockOrgCreditFetch({
      creditData: { ...credit, welcome_remaining_cents: 0, welcome_credit_expires_at: "2020-01-01T00:00:00Z" },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-trial-target")).toBeDisabled();
    expect(screen.getByTestId("admin-org-credit-trial-save")).toBeDisabled();
    expect(screen.getByText(ADMIN_NO_ACTIVE_TRIAL_COPY)).toBeInTheDocument();
  });

  it("reports a stale balance and reloads", async () => {
    const user = userEvent.setup();
    const fetchMock = mockOrgCreditFetch({
      onPutBalance: () => new Response("The balance changed. Reload and try again.", { status: 409 }),
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    const grant = await screen.findByTestId("admin-org-credit-grant-target");
    await user.clear(grant);
    await user.type(grant, "50");
    await user.click(screen.getByTestId("admin-org-credit-grant-save"));

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith(CREDIT_BALANCE_CHANGED_COPY));
    await waitFor(() => expect(screen.getByTestId("admin-org-credit-grant-target")).toHaveValue("20.00"));
    const creditLoads = fetchMock.mock.calls.filter(([input]) => String(input).endsWith("/llm-credit"));
    expect(creditLoads.length).toBe(2);
  });
});
