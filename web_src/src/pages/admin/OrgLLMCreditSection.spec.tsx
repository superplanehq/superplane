import {
  credit,
  catalog,
  grants,
  billingPlan,
  jsonResponse,
  mockOrgCreditFetch,
} from "./OrgLLMCreditSection.test-support";
import {
  ADMIN_POLAR_MANAGED_PLAN_COPY,
  ADMIN_LOCAL_PLAN_COPY,
  ADMIN_PLAN_UNKNOWN_COPY,
  ADMIN_TRIAL_ENDS_LABEL,
  ADMIN_TRIAL_ENDS_HELP,
  ADMIN_TRIAL_ENDS_ERROR,
} from "./OrgBillingPlanField";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  ADMIN_CREDIT_HISTORY_ERROR_COPY,
  ADMIN_NO_ACTIVE_TRIAL_COPY,
  OrgLLMCreditSection,
} from "./OrgLLMCreditSection";
import { CREDIT_BALANCE_CHANGED_COPY, prefilledTrialEndDate } from "./useOrgLLMCredit";
import { showErrorToast } from "@/lib/toast";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

let billingProductEnabled = true;

vi.mock("@/hooks/useBillingProductEnabled", () => ({
  useBillingProductEnabled: () => billingProductEnabled,
}));

function futureUtcDate(daysAhead: number): string {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + daysAhead))
    .toISOString()
    .slice(0, 10);
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.mocked(showErrorToast).mockClear();
  billingProductEnabled = true;
});

describe("OrgLLMCreditSection billing plan", () => {
  it("disables plans that require payment and saves a manual plan", async () => {
    const fetchMock = mockOrgCreditFetch();
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);

    await user.click(await screen.findByTestId("admin-org-billing-plan"));
    expect(await screen.findByRole("option", { name: "Business" })).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByRole("option", { name: "Pay-as-you-go" }).getAttribute("aria-disabled")).toBe("true");
    await user.click(screen.getByRole("option", { name: "Business Manual" }));
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));

    await waitFor(() => {
      const put = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT");
      expect(String(put?.[0])).toBe("/admin/api/organizations/org-1/billing-plan");
      expect(JSON.parse(String(put?.[1]?.body))).toEqual({ plan: "business_manual" });
    });
  });

  it("shows the assigned catalog key when the legacy plan field differs", async () => {
    mockOrgCreditFetch({ plan: { ...billingPlan(false, null, "business"), catalog_key: "business_manual" } });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toHaveTextContent("Business Manual");
  });

  it("blocks plan changes when the catalog cannot load", async () => {
    mockOrgCreditFetch({ catalogStatus: 500 });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load billing plans");
    expect(screen.getByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.getByTestId("admin-org-billing-plan-save")).toBeDisabled();
  });

  it("keeps Save plan when Polar is not set up", async () => {
    mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeEnabled();
    expect(screen.getByTestId("admin-org-billing-plan-save")).toBeInTheDocument();
    expect(screen.getByText(ADMIN_LOCAL_PLAN_COPY)).toBeInTheDocument();
  });

  it("hides plan controls when product billing is off", async () => {
    billingProductEnabled = false;
    const fetchMock = mockOrgCreditFetch({ planStatus: 404 });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-total-remaining")).toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-billing-plan")).not.toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.some(([input]) => String(input).includes("/billing-plan"))).toBe(false);
    expect(showErrorToast).not.toHaveBeenCalledWith("Failed to load billing plan");
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

  it("shows the trial end and date field only for Trial", async () => {
    const trialDate = futureUtcDate(40);
    const trialEndsAt = `${trialDate}T23:59:59Z`;
    mockOrgCreditFetch({ plan: billingPlan(false, trialEndsAt, "trial") });
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-trial-ends-input")).toHaveValue(trialDate);
    expect(screen.getByText(ADMIN_TRIAL_ENDS_LABEL)).toBeInTheDocument();
    expect(screen.getByText(ADMIN_TRIAL_ENDS_HELP)).toBeInTheDocument();

    await user.click(screen.getByTestId("admin-org-billing-plan"));
    expect((await screen.findByRole("option", { name: /^Business$/ })).getAttribute("aria-disabled")).toBe("true");
    await user.click(await screen.findByRole("option", { name: "Business Manual" }));

    expect(screen.queryByTestId("admin-org-trial-ends-input")).not.toBeInTheDocument();
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_LABEL)).not.toBeInTheDocument();
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_HELP)).not.toBeInTheDocument();
  });

  it("hides a leftover trial end on an assigned paid plan", async () => {
    mockOrgCreditFetch({ plan: billingPlan(false, `${futureUtcDate(40)}T23:59:59Z`, "business") });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toHaveTextContent("Business Manual");
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_LABEL)).not.toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-trial-ends-input")).not.toBeInTheDocument();
  });

  it("hides the trial end when Polar manages billing", async () => {
    mockOrgCreditFetch({ plan: billingPlan(true, `${futureUtcDate(40)}T23:59:59Z`, "business") });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.queryByTestId("admin-org-trial-ends-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-trial-ends")).not.toBeInTheDocument();
    expect(screen.queryByText(ADMIN_TRIAL_ENDS_LABEL)).not.toBeInTheDocument();
  });

  it("prefills 14 days from today when no future trial end exists", async () => {
    mockOrgCreditFetch();
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
        if (url.endsWith("/feature-definitions")) return jsonResponse([]);
        if (url.endsWith("/features")) return jsonResponse({ grants: [], effective: { features: {} } });
        if (url.endsWith("/billing-plans"))
          return jsonResponse([
            { key: "trial", name: "Trial", category: "trial" },
            { key: "business_legacy", name: "Business", category: "paid" },
            { key: "business_manual", name: "Business Manual", category: "paid" },
            { key: "none", name: "None", category: "none" },
          ]);
        if (url.endsWith("/feature-definitions")) return jsonResponse([]);
        if (url.endsWith("/features")) return jsonResponse({ grants: [], effective: { features: {} } });
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
        if (url.endsWith("/billing-plans")) return jsonResponse(catalog);
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

  it("edits the trial end and unlocks Set trial credit for a custom trial plan", async () => {
    const trialEndsAt = `${futureUtcDate(30)}T23:59:59Z`;
    const requests: Array<{ url: string; body: string | null }> = [];
    const customTrial = { key: "custom_trial", name: "Custom Trial", category: "trial", require_payment_method: false };
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.endsWith("/feature-definitions")) return jsonResponse([]);
        if (url.endsWith("/features")) return jsonResponse({ grants: [], effective: { features: {} } });
        if (url.endsWith("/llm-credit/grants")) return jsonResponse({ grants });
        if (url.endsWith("/llm-credit")) {
          return jsonResponse({
            ...credit,
            welcome_remaining_cents: 0,
            welcome_credit_expires_at: "2020-01-01T00:00:00Z",
          });
        }
        if (url.endsWith("/billing-plans")) return jsonResponse([...catalog, customTrial]);
        if (url.includes("/billing-plan")) {
          if (init?.method === "PUT") {
            requests.push({ url, body: typeof init.body === "string" ? init.body : null });
          }
          return jsonResponse(billingPlan(false, trialEndsAt, "custom_trial", "trial"));
        }
        return new Response("not found", { status: 404 });
      }),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-billing-plan")).toHaveTextContent("Custom Trial");
    expect(screen.getByTestId("admin-org-credit-trial-target")).toBeEnabled();
    expect(screen.queryByText(ADMIN_NO_ACTIVE_TRIAL_COPY)).not.toBeInTheDocument();

    const input = await screen.findByTestId("admin-org-trial-ends-input");
    expect(input).toHaveValue(futureUtcDate(30));
    const newDate = futureUtcDate(45);
    fireEvent.change(input, { target: { value: newDate } });
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));
    await waitFor(() => expect(requests).toHaveLength(1));
    expect(JSON.parse(requests[0]?.body ?? "{}")).toEqual({
      plan: "custom_trial",
      trial_ends_at: `${newDate}T23:59:59Z`,
    });
  });
});

describe("OrgLLMCreditSection balances", () => {
  it("shows credit balances and history without controls to set included credit", async () => {
    mockOrgCreditFetch();
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-total-remaining")).toHaveTextContent("$95.00");
    expect(screen.getByTestId("admin-org-credit-trial-remaining")).toHaveTextContent("$45.00");
    expect(screen.getByTestId("admin-org-credit-included-remaining")).toHaveTextContent("$0.00");
    expect(screen.queryByLabelText("Included credit (USD)")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Set included credit" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Included credit expiration (UTC)")).not.toBeInTheDocument();
    expect(screen.getByTestId("admin-org-credit-topup-remaining")).toHaveTextContent("$30.00");
    expect(screen.getByTestId("admin-org-credit-grant-remaining")).toHaveTextContent("$20.00");
    expect(screen.getByTestId("admin-org-credit-trial-target")).toHaveValue("45.00");
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();

    const history = screen.getByTestId("admin-org-credit-history");
    expect(await within(history).findByText("Signup credit adjustment")).toBeInTheDocument();
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
    expect(screen.getByTestId("admin-org-credit-trial-change")).toHaveTextContent("Balance change: -$35.00");
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
      plan: billingPlan(false, "2020-01-01T00:00:00Z", "none"),
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-trial-target")).toBeDisabled();
    expect(screen.getByTestId("admin-org-credit-trial-save")).toBeDisabled();
    expect(screen.getByText(ADMIN_NO_ACTIVE_TRIAL_COPY)).toBeInTheDocument();
  });

  it("enables Set trial credit on an open Trial when remaining trial credit is zero", async () => {
    const user = userEvent.setup();
    const trialEndsAt = new Date(Date.now() + 10 * 24 * 60 * 60 * 1000).toISOString();
    const saved = {
      ...credit,
      remaining_credit_cents: 0,
      welcome_remaining_cents: 0,
      welcome_credit_expires_at: null,
    };
    const updated = {
      ...saved,
      remaining_credit_cents: 2500,
      welcome_remaining_cents: 2500,
      welcome_credit_expires_at: trialEndsAt,
    };
    const fetchMock = mockOrgCreditFetch({
      creditData: saved,
      plan: billingPlan(false, trialEndsAt, "trial"),
      onPutBalance: () => jsonResponse(updated),
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    const trial = await screen.findByTestId("admin-org-credit-trial-target");
    expect(trial).toBeEnabled();
    expect(screen.getByTestId("admin-org-credit-trial-save")).toBeDisabled();
    expect(screen.queryByText(ADMIN_NO_ACTIVE_TRIAL_COPY)).not.toBeInTheDocument();
    expect(screen.getByText("No trial credit.")).toBeInTheDocument();

    await user.clear(trial);
    await user.type(trial, "25");
    await user.click(screen.getByTestId("admin-org-credit-trial-save"));

    await waitFor(() => {
      const put = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT");
      expect(put).toBeDefined();
      expect(String(put?.[0])).toBe("/admin/api/organizations/org-1/llm-credit/balances");
      expect(JSON.parse(String(put?.[1]?.body))).toEqual({
        bucket: "trial",
        target_cents: 2500,
        expected_remaining_cents: 0,
        note: "",
      });
    });
    expect(screen.getByTestId("admin-org-credit-trial-remaining")).toHaveTextContent("$25.00");
    expect(screen.queryByText("No trial credit.")).not.toBeInTheDocument();
    expect(screen.getByText(/Expires on /)).toBeInTheDocument();
  });

  it("keeps trial credit locked when only the unsaved plan is an open Trial", async () => {
    const user = userEvent.setup();
    mockOrgCreditFetch({
      creditData: { ...credit, welcome_remaining_cents: 0, welcome_credit_expires_at: null },
      plan: billingPlan(false, null, "none"),
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-trial-target")).toBeDisabled();
    await user.click(screen.getByTestId("admin-org-billing-plan"));
    await user.click(await screen.findByRole("option", { name: "Trial" }));
    expect(screen.getByTestId("admin-org-trial-ends-input")).toBeInTheDocument();
    expect(screen.getByTestId("admin-org-credit-trial-target")).toBeDisabled();
    expect(screen.getByText(ADMIN_NO_ACTIVE_TRIAL_COPY)).toBeInTheDocument();
  });

  it("keeps trial credit locked when Polar manages the saved Trial plan", async () => {
    const trialEndsAt = new Date(Date.now() + 10 * 24 * 60 * 60 * 1000).toISOString();
    mockOrgCreditFetch({
      creditData: { ...credit, welcome_remaining_cents: 0, welcome_credit_expires_at: null },
      plan: { ...billingPlan(true, trialEndsAt, "trial"), polar_managed: true },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByTestId("admin-org-credit-trial-target")).toBeDisabled();
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

  it("shows a load error when the credit history request fails", async () => {
    const user = userEvent.setup();
    let grantsFailed = true;
    mockOrgCreditFetch({
      onGetGrants: () => (grantsFailed ? new Response("failed", { status: 500 }) : jsonResponse({ grants })),
    });
    render(<OrgLLMCreditSection orgId="org-1" />);

    expect(await screen.findByText(ADMIN_CREDIT_HISTORY_ERROR_COPY)).toBeInTheDocument();
    expect(screen.queryByText("No credit grants yet.")).not.toBeInTheDocument();

    grantsFailed = false;
    const history = screen.getByTestId("admin-org-credit-history");
    await user.click(within(history).getByRole("button", { name: "Try again" }));
    expect(await within(history).findByText("Signup credit adjustment")).toBeInTheDocument();
    expect(within(history).queryByText(ADMIN_CREDIT_HISTORY_ERROR_COPY)).not.toBeInTheDocument();
  });
});
