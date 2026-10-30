import { billingPlan, jsonResponse, mockOrgCreditFetch } from "./OrgLLMCreditSection.test-support";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { OrgLLMCreditSection } from "./OrgLLMCreditSection";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn(), showSuccessToast: vi.fn() }));
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));
afterEach(() => vi.unstubAllGlobals());

describe("OrgLLMCreditSection Polar verification", () => {
  it("places billing diagnostics after credit history and keeps them collapsed until opened", async () => {
    mockOrgCreditFetch({ plan: billingPlan(true) });
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    const toggle = screen.getByRole("button", { name: "Billing diagnostics" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Stored billing data")).not.toBeInTheDocument();
    const history = await screen.findByTestId("admin-org-credit-history");
    expect(history.compareDocumentPosition(toggle) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(await screen.findByText("Stored billing data")).toBeVisible();
    await user.click(toggle);
    expect(screen.queryByText("Stored billing data")).not.toBeInTheDocument();
  });

  it.each([
    ["usable", "Usable card found"],
    ["absent", "No usable card (absent or expired)"],
    ["failed", "Failed"],
  ])("shows the distinct %s card result and recorded evidence", async (cardStatus, expected) => {
    mockOrgCreditFetch({
      plan: {
        ...billingPlan(false),
        record_exists: false,
        plan: "none",
        catalog_key: "none",
        workspaces: [
          { id: "ws-1", name: "Engineering", onboarding_completed_at: "2026-10-21T10:00:00Z" },
          { id: "ws-2", name: "Support", onboarding_completed_at: null },
        ],
        checkout_attempts: [
          {
            id: "checkout",
            plan_key: "team",
            state: "failed",
            polar_checkout_id: "polar-checkout",
            polar_status: "failed",
            last_error: "Payment declined",
            created_at: "2026-10-27T11:00:00Z",
            updated_at: "2026-10-27T11:02:00Z",
            expires_at: null,
            resolved_at: "2026-10-27T11:02:00Z",
          },
        ],
        polar: { ...billingPlan(true).polar!, card_status: cardStatus },
      },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    const panel = within(screen.getByRole("region", { name: "Billing diagnostics" }));
    expect(await panel.findByText("No billing-plan record")).toBeVisible();
    const card = panel.getByText("Usable saved card").parentElement!;
    expect(within(card).getByText(expected)).toBeVisible();
    expect(panel.getByText("2026-10-21 10:00:00 UTC")).toBeVisible();
    expect(panel.getByText(/Support.*Not completed/)).toBeVisible();
    expect(panel.getByText("Payment declined")).toBeVisible();
    expect(panel.getByText("2026-10-27 11:00:00 UTC")).toBeVisible();
    expect(panel.getByText(/Historical card changes: Unknown/)).toBeVisible();
  });

  it("checks Polar on opening credits, retries failures, and refreshes stored diagnostics after save", async () => {
    const initial = {
      ...billingPlan(false),
      polar_subscription_verification: "not_checked" as const,
      polar: {
        ...billingPlan(true).polar!,
        checked_at: null,
        customer_status: "not_checked",
        card_status: "not_checked",
        subscription_verification: "not_checked",
      },
    };
    const fetchMock = mockOrgCreditFetch({ plan: initial });
    const original = fetchMock.getMockImplementation()!;
    let checks = 0;
    let saved = false;
    fetchMock.mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.includes("check_polar=true")) {
        checks++;
        if (checks === 1) return new Response("upstream failed", { status: 502 });
        return jsonResponse({
          ...initial,
          polar_subscription_verification: "none",
          polar: {
            ...initial.polar,
            checked_at: "2026-10-28T12:00:00Z",
            customer_status: "failed",
            customer_error: "Customer lookup failed. Try again.",
            card_status: "unknown",
            card_error: "Card status is unknown.",
            subscription_verification: "none",
          },
        });
      }
      if (url.endsWith("/billing-plan") && init?.method === "PUT") saved = true;
      if (url.endsWith("/billing-plan") && saved)
        return jsonResponse({ ...initial, plan_source: "admin", catalog_key: "business_manual" });
      return original(input, init);
    });
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await waitFor(() => expect(checks).toBe(1));
    expect(screen.getByRole("button", { name: "Billing diagnostics" })).toHaveAttribute("aria-expanded", "false");
    await user.click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    const panel = within(screen.getByRole("region", { name: "Billing diagnostics" }));
    expect(await panel.findByRole("alert")).toHaveTextContent("Could not load billing diagnostics");
    expect(screen.getByTestId("admin-org-billing-plan")).toBeDisabled();
    await user.click(panel.getByRole("button", { name: "Try again" }));
    expect(await panel.findByText("Card status is unknown.")).toBeVisible();
    expect(panel.queryByText(/No usable card/)).not.toBeInTheDocument();
    expect(panel.getByText("2026-10-28 12:00:00 UTC")).toBeVisible();
    expect(screen.getByTestId("admin-org-billing-plan")).toBeEnabled();
    await user.click(screen.getByTestId("admin-org-billing-plan"));
    await user.click(await screen.findByRole("option", { name: "Business Manual" }));
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));
    await waitFor(() => expect(panel.getByText("admin")).toBeVisible());
    expect(checks).toBe(2);
    expect(screen.getByTestId("admin-org-billing-plan")).toBeDisabled();
  });

  it("clears diagnostic results and ignores an old check when organizations change", async () => {
    const initial = {
      ...billingPlan(false),
      polar_subscription_verification: "not_checked" as const,
      polar: { ...billingPlan(true).polar!, checked_at: null },
    };
    const fetchMock = mockOrgCreditFetch({ plan: initial });
    const original = fetchMock.getMockImplementation()!;
    let finishCheck: (response: Response) => void = () => {};
    fetchMock.mockImplementation(async (input, init) => {
      if (String(input).includes("/org-1/billing-plan?check_polar=true"))
        return new Promise<Response>((resolve) => {
          finishCheck = resolve;
        });
      return original(input, init);
    });
    const view = render(<OrgLLMCreditSection orgId="org-1" />);
    expect(screen.getByRole("region", { name: "Billing diagnostics" })).toHaveAttribute("aria-busy", "true");
    const oldRequest = fetchMock.mock.calls.find(([input]) => String(input).includes("check_polar=true"));
    view.rerender(<OrgLLMCreditSection orgId="org-2" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(oldRequest?.[1]?.signal?.aborted).toBe(true);
    finishCheck(
      jsonResponse({
        ...initial,
        polar_subscription_verification: "none",
        polar: { ...initial.polar, customer_id: "old-customer", checked_at: new Date().toISOString() },
      }),
    );
    await waitFor(() => expect(screen.getByRole("button", { name: "Check Polar" })).toBeEnabled());
    expect(screen.queryByText("old-customer")).not.toBeInTheDocument();
    expect(screen.getByTestId("admin-org-billing-plan")).toBeDisabled();
  });

  it("enables stale plans, shows the stored subscription link, and clears the warning after save", async () => {
    const saved = {
      ...billingPlan(true),
      polar_subscription_verification: "none" as const,
      polar_subscription_id: "sub_stored",
      polar_subscription_url: "https://polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_stored",
    };
    const fetchMock = mockOrgCreditFetch({ plan: saved });
    const original = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation(async (input, init) => {
      if (init?.method === "PUT") {
        saved.polar_managed = false;
        saved.polar_subscription_status = "";
        saved.catalog_key = "business_manual";
        saved.plan_source = "admin";
      }
      return original(input, init);
    });
    const user = userEvent.setup();
    const view = render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(await screen.findByTestId("admin-org-billing-plan")).toBeEnabled();
    const warning = "No active Polar subscription";
    expect(screen.getByText(warning)).toBeInTheDocument();
    expect(screen.getAllByText("sub_stored").length).toBeGreaterThan(0);
    expect(screen.getByRole("link", { name: "View subscription in Polar" })).toHaveAttribute(
      "href",
      saved.polar_subscription_url,
    );
    await user.click(screen.getByTestId("admin-org-billing-plan"));
    await user.click(await screen.findByRole("option", { name: "Business Manual" }));
    expect(screen.getByText(warning)).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-org-billing-plan-save"));
    await waitFor(() => expect(screen.queryByText(warning)).not.toBeInTheDocument());
    view.unmount();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(await screen.findByTestId("admin-org-billing-plan")).toBeEnabled();
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  it.each([
    ["business_legacy", true],
    ["business_manual", false],
    ["pay_as_you_go", false],
    ["trial", false],
  ] as const)("warns only when the saved %s plan requires a subscription", async (key, warns) => {
    mockOrgCreditFetch({
      plan: { ...billingPlan(false), catalog_key: key, polar_subscription_verification: "none" },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    await screen.findByTestId("admin-org-billing-plan");
    const warning = "No active Polar subscription";
    if (warns) {
      expect(screen.getByText(warning)).toBeInTheDocument();
      return;
    }
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  it("locks editing when verification fails even for a local plan", async () => {
    mockOrgCreditFetch({ plan: { ...billingPlan(false), polar_subscription_verification: "failed" } });
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(await screen.findByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(
      screen.getByText("SuperPlane could not verify the Polar subscription. Select Check Polar to try again."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("admin-org-billing-plan-save")).not.toBeInTheDocument();
  });

  it("shows canceled subscription IDs without a link when no dashboard URL is configured", async () => {
    mockOrgCreditFetch({
      plan: { ...billingPlan(false), polar_subscription_status: "canceled", polar_subscription_id: "sub_canceled" },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect((await screen.findAllByText("sub_canceled")).length).toBeGreaterThan(0);
    expect(screen.queryByRole("link", { name: "View subscription in Polar" })).not.toBeInTheDocument();
  });

  it("does not show a subscription link without a subscription ID", async () => {
    mockOrgCreditFetch({
      plan: {
        ...billingPlan(false),
        polar_subscription_url: "https://polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_missing",
      },
    });
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    await screen.findByTestId("admin-org-billing-plan");
    expect(screen.queryByRole("link", { name: "View subscription in Polar" })).not.toBeInTheDocument();
  });
});
