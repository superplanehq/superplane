import { billingPlan, jsonResponse, mockOrgCreditFetch } from "./OrgLLMCreditSection.test-support";
import type { OrganizationBillingPlan } from "./useOrgLLMCredit";
import { OrgLLMCreditSection } from "./OrgLLMCreditSection";
import { showSuccessToast } from "@/lib/toast";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn(), showSuccessToast: vi.fn() }));
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));

afterEach(() => {
  vi.unstubAllGlobals();
  vi.mocked(showSuccessToast).mockClear();
});

function checkedPlan(): OrganizationBillingPlan {
  return {
    ...billingPlan(false),
    polar_customer_id: "customer_stored",
    polar_subscription_verification: "none",
    polar: {
      ...billingPlan(true).polar!,
      customer_id: "customer_live",
      customer_id_mismatch: true,
      subscription_verification: "none",
    },
  };
}

function mockCustomerSync(
  initial: OrganizationBillingPlan,
  onSync: () => Response | Promise<Response>,
  onCheck?: () => Response,
) {
  const fetchMock = mockOrgCreditFetch({ plan: initial });
  const original = fetchMock.getMockImplementation()!;
  fetchMock.mockImplementation(async (input, init) => {
    const url = String(input);
    if (url.endsWith("/billing-customer/sync")) return onSync();
    if (url.includes("check_polar=true") && onCheck) return onCheck();
    return original(input, init);
  });
  return fetchMock;
}

describe("OrgLLMCreditSection customer sync", () => {
  it("checks before explicit sync, locks conflicting actions, and refreshes the stored customer", async () => {
    const checked = checkedPlan();
    const initial = {
      ...checked,
      polar_subscription_verification: "not_checked" as const,
      polar: { ...checked.polar!, checked_at: null, customer_status: "not_checked", card_status: "not_checked" },
    };
    let finishSync: (response: Response) => void = () => {};
    const fetchMock = mockCustomerSync(
      initial,
      () =>
        new Promise<Response>((resolve) => {
          finishSync = resolve;
        }),
      () => jsonResponse(checked),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    const panel = within(screen.getByRole("region", { name: "Billing diagnostics" }));
    await waitFor(() => expect(panel.getByRole("button", { name: "Sync customer from Polar" })).toBeEnabled());
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(false);

    await user.click(panel.getByRole("button", { name: "Sync customer from Polar" }));
    expect(panel.getByRole("button", { name: "Syncing customer..." })).toBeDisabled();
    expect(panel.getByRole("button", { name: "Check Polar" })).toBeDisabled();
    expect(screen.getByTestId("admin-org-billing-plan")).toBeDisabled();
    expect(screen.getByTestId("admin-org-billing-plan-save")).toBeDisabled();
    const sync = fetchMock.mock.calls.find(([input]) => String(input).endsWith("/billing-customer/sync"));
    expect(String(sync?.[0])).toBe("/admin/api/organizations/org-1/billing-customer/sync");
    expect(sync?.[1]).toMatchObject({ method: "POST", credentials: "include" });

    finishSync(
      jsonResponse({
        billing_plan: {
          ...checked,
          polar_customer_id: "customer_live",
          polar: { ...checked.polar!, customer_id_mismatch: false, checked_at: "2026-10-28T13:00:00Z" },
        },
      }),
    );
    await waitFor(() => expect(panel.getByRole("button", { name: "Sync customer from Polar" })).toBeEnabled());
    expect(within(panel.getByText("Stored Polar customer ID").parentElement!).getByText("customer_live")).toBeVisible();
    expect(panel.getByText("2026-10-28 13:00:00 UTC")).toBeVisible();
    expect(panel.queryByText("The live customer ID differs from the stored ID.")).not.toBeInTheDocument();
    expect(screen.getByTestId("admin-org-billing-plan")).toBeEnabled();
    expect(showSuccessToast).toHaveBeenCalledWith("Polar customer synced");
  });

  it.each([
    ["found", "absent"],
    ["found", "failed"],
    ["absent", "unknown"],
    ["failed", "unknown"],
  ])("disables sync when the customer is %s and card is %s", async (customerStatus, cardStatus) => {
    const checked = checkedPlan();
    const fetchMock = mockCustomerSync(
      { ...checked, polar: { ...checked.polar!, customer_status: customerStatus, card_status: cardStatus } },
      () => jsonResponse({ billing_plan: checked }),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    const sync = await screen.findByRole("button", { name: "Sync customer from Polar" });
    expect(sync).toBeDisabled();
    await user.click(sync);
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(false);
  });

  it("applies fresh diagnostics when a card disappears before sync", async () => {
    const checked = checkedPlan();
    mockCustomerSync(checked, () =>
      jsonResponse(
        {
          billing_plan: {
            ...checked,
            polar: { ...checked.polar!, card_status: "absent", checked_at: "2026-10-28T14:00:00Z" },
          },
          error: "No usable saved card found in Polar.",
        },
        422,
      ),
    );
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    await user.click(await screen.findByRole("button", { name: "Sync customer from Polar" }));
    const panel = within(screen.getByRole("region", { name: "Billing diagnostics" }));
    expect(await panel.findByText("No usable saved card found in Polar.")).toBeVisible();
    expect(panel.getByText("No usable card (absent or expired)")).toBeVisible();
    expect(panel.getByText("2026-10-28 14:00:00 UTC")).toBeVisible();
    expect(
      within(panel.getByText("Stored Polar customer ID").parentElement!).getByText("customer_stored"),
    ).toBeVisible();
    expect(panel.getByRole("button", { name: "Sync customer from Polar" })).toBeDisabled();
    expect(panel.getByRole("button", { name: "Check Polar" })).toBeEnabled();
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it.each(["JSON", "plain text"])("keeps diagnostics visible and retries sync after a %s failure", async (format) => {
    const checked = checkedPlan();
    let attempts = 0;
    mockCustomerSync(checked, () => {
      attempts++;
      if (attempts === 1) {
        if (format === "plain text") return new Response("upstream failed", { status: 502 });
        return jsonResponse({ billing_plan: checked, error: "Could not sync the Polar customer. Try again." }, 502);
      }
      return jsonResponse({ billing_plan: { ...checked, polar_customer_id: "customer_live" } });
    });
    const user = userEvent.setup();
    render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    await user.click(await screen.findByRole("button", { name: "Sync customer from Polar" }));
    const panel = within(screen.getByRole("region", { name: "Billing diagnostics" }));
    expect(await panel.findByText("Could not sync the Polar customer. Try again.")).toBeVisible();
    expect(panel.getByText("Stored billing data")).toBeVisible();
    expect(panel.getByText("Usable card found")).toBeVisible();
    await user.click(panel.getByRole("button", { name: "Sync customer from Polar" }));
    await waitFor(() =>
      expect(panel.queryByText("Could not sync the Polar customer. Try again.")).not.toBeInTheDocument(),
    );
    await waitFor(() => expect(showSuccessToast).toHaveBeenCalledWith("Polar customer synced"));
    expect(attempts).toBe(2);
  });

  it("aborts old organization syncs and ignores their diagnostics and success toast", async () => {
    const checked = checkedPlan();
    let finishSync: (response: Response) => void = () => {};
    const fetchMock = mockCustomerSync(
      checked,
      () =>
        new Promise<Response>((resolve) => {
          finishSync = resolve;
        }),
    );
    const user = userEvent.setup();
    const view = render(<OrgLLMCreditSection orgId="org-1" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    await user.click(await screen.findByRole("button", { name: "Sync customer from Polar" }));
    const sync = fetchMock.mock.calls.find(([input]) => String(input).endsWith("/billing-customer/sync"));
    view.rerender(<OrgLLMCreditSection orgId="org-2" />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Billing diagnostics" }));
    expect(sync?.[1]?.signal?.aborted).toBe(true);
    finishSync(jsonResponse({ billing_plan: { ...checked, polar_customer_id: "old-org-customer" } }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Sync customer from Polar" })).toBeEnabled());
    expect(screen.queryByText("old-org-customer")).not.toBeInTheDocument();
    expect(showSuccessToast).not.toHaveBeenCalled();
  });
});
