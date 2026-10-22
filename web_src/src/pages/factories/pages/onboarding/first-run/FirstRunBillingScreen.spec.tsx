import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { FirstRunBillingScreen } from "./FirstRunBillingScreen";

let billing: OrganizationsDescribeOrganizationBillingResponse;
const describeBilling = vi.fn(async () => ({ data: billing }));
const selectPlan = vi.fn(async ({ body }: { body: { planKey: string } }) => {
  if (body.planKey === "pro") return { data: { checkoutUrl: "https://polar.sh/checkout/pro" } };
  billing = { ...billing, catalogKey: body.planKey, requirePaymentMethod: body.planKey === "payg" };
  return { data: {} };
});
const createSession = vi.fn(async () => ({
  data: {
    sessionToken: "polar_cst_setup",
    portalUrl: "https://sandbox.polar.sh/acme/portal/overview?customer_session_token=test-session",
  },
}));
const syncBilling = vi.fn(async () => ({ data: billing }));
vi.mock("@/api-client", () => ({
  organizationsDescribeOrganizationBilling: describeBilling,
  organizationsSelectBillingPlan: selectPlan,
  organizationsCreatePaymentMethodSession: createSession,
  organizationsSyncOrganizationBilling: syncBilling,
}));

function renderBilling(path = "/org-1/workspaces/pay/setup?step=billing") {
  const onContinue = vi.fn(async () => {});
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <FirstRunBillingScreen
          organizationId="org-1"
          chrome={{ stepIndex: 5 }}
          saving={false}
          onContinue={onContinue}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onContinue;
}

beforeEach(() => {
  billing = {
    billingEnabled: true,
    hasPaymentMethod: false,
    catalogKey: "trial",
    availablePlans: [
      { key: "free", name: "Free", priceCents: "0" },
      { key: "payg", name: "Pay-as-you-go", priceCents: "0", requirePaymentMethod: true },
      { key: "team", name: "Team", priceCents: "3000", perSeat: true, subscriptionRequired: true },
      { key: "pro", name: "Pro", priceCents: "2000", requirePaymentMethod: true, subscriptionRequired: true },
    ],
  };
  describeBilling.mockClear();
  selectPlan.mockClear();
  createSession.mockClear();
  syncBilling.mockClear();
});

it("keeps setup blocked when plans cannot load", async () => {
  describeBilling.mockRejectedValueOnce(new Error("Billing unavailable"));
  const finish = renderBilling();
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not load plans.");
  expect(screen.queryByRole("button", { name: /None/ })).toBeNull();
  expect(screen.getByRole("button", { name: "Finish setup" })).toBeDisabled();
  expect(selectPlan).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
});

it("activates a plan without a card or checkout when neither is required", async () => {
  const user = userEvent.setup();
  const finish = renderBilling();
  expect(screen.queryByRole("button", { name: "Check payment status" })).toBeNull();
  await user.click(await screen.findByRole("button", { name: /Free/ }));
  await user.click(screen.getByRole("button", { name: "Finish setup" }));
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  expect(selectPlan).toHaveBeenCalledWith(
    expect.objectContaining({
      body: { planKey: "free", returnPath: "/org-1/workspaces/pay/setup?step=billing&plan=free" },
    }),
  );
  expect(createSession).not.toHaveBeenCalled();
});

it.each(["payg", "pro"])("verifies the saved card before finishing plan %s", async (planKey) => {
  if (planKey === "pro") {
    billing = { ...billing, catalogKey: "pro", planSource: "polar", creditPurchaseAllowed: true };
  }
  const user = userEvent.setup();
  const finish = renderBilling();
  const previousOpen = window.open;
  const popup = { closed: false, opener: {}, location: { replace: vi.fn() }, close: vi.fn() };
  window.open = vi.fn(() => popup as unknown as Window);
  try {
    await user.click(await screen.findByRole("button", { name: planKey === "payg" ? /Pay-as-you-go/ : /Pro/ }));
    await waitFor(() =>
      expect(popup.location.replace).toHaveBeenCalledWith(
        "https://sandbox.polar.sh/acme/portal/settings?customer_session_token=test-session",
      ),
    );
    expect(popup.opener).toBeNull();
  } finally {
    window.open = previousOpen;
  }
  expect(screen.queryByRole("button", { name: "Add payment method" })).not.toBeInTheDocument();
  expect(await screen.findByText("Waiting for your saved card…")).toBeInTheDocument();
  expect(document.querySelector("iframe")).toBeNull();
  expect(selectPlan).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
  expect(popup.close).not.toHaveBeenCalled();
  expect(screen.getByRole("button", { name: "Finish setup" })).toBeDisabled();
  billing = { ...billing, hasPaymentMethod: true };
  await waitFor(() => expect(popup.close).toHaveBeenCalled(), { timeout: 4000 });
  await waitFor(() => expect(screen.getByRole("button", { name: "Finish setup" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "Finish setup" }));
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  expect(selectPlan).toHaveBeenCalledTimes(planKey === "payg" ? 1 : 0);
  if (planKey === "payg") {
    expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: expect.objectContaining({ planKey }) }));
  }
});

it("lets the user retry when the browser blocks the card setup window", async () => {
  const user = userEvent.setup();
  renderBilling();
  const previousOpen = window.open;
  window.open = vi.fn(() => null);
  try {
    await user.click(await screen.findByRole("button", { name: /Pay-as-you-go/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Allow pop-ups for SuperPlane, then try again.");
    expect(createSession).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Add payment method" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Pay-as-you-go/ })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: /Pay-as-you-go/ }));
    expect(window.open).toHaveBeenCalledTimes(2);
    expect(screen.getByRole("button", { name: "Finish setup" })).toBeDisabled();
  } finally {
    window.open = previousOpen;
  }
});

it("sends subscription plans to checkout with an onboarding return path", async () => {
  const user = userEvent.setup();
  const finish = renderBilling();
  const previousAssign = window.location.assign;
  const assign = vi.fn();
  window.location.assign = assign;
  try {
    await user.click(await screen.findByRole("button", { name: /Pro/ }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.sh/checkout/pro"));
    expect(createSession).not.toHaveBeenCalled();
    expect(finish).not.toHaveBeenCalled();
  } finally {
    window.location.assign = previousAssign;
  }
});

it("syncs a returning subscription and only continues when it is active", async () => {
  billing = { ...billing, catalogKey: "pro", planSource: "polar", creditPurchaseAllowed: true, hasPaymentMethod: true };
  const user = userEvent.setup();
  const finish = renderBilling("/org-1/workspaces/pay/setup?step=billing&subscribed=1");
  await waitFor(() => expect(syncBilling).toHaveBeenCalledTimes(1));
  await user.click(await screen.findByRole("button", { name: /Pro/ }));
  await user.click(screen.getByRole("button", { name: "Finish setup" }));
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  expect(selectPlan).not.toHaveBeenCalled();
});

it("waits for a returning subscription without creating another checkout", async () => {
  const user = userEvent.setup();
  const finish = renderBilling("/org-1/workspaces/pay/setup?step=billing&plan=pro&subscribed=1");
  await waitFor(() => expect(syncBilling).toHaveBeenCalledTimes(1));
  await user.click(await screen.findByRole("button", { name: "Subscribe to Pro" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("The plan is not active yet.");
  expect(selectPlan).not.toHaveBeenCalled();
  expect(finish).not.toHaveBeenCalled();
});

it("finishes setup with an active private plan that is not listed", async () => {
  billing = {
    ...billing,
    catalogKey: "enterprise",
    planName: "Enterprise",
    planCategory: "paid",
    planPriceCents: "5000",
    planSource: "polar",
    creditPurchaseAllowed: true,
    hasPaymentMethod: true,
  };
  const user = userEvent.setup();
  const finish = renderBilling();
  const current = await screen.findByRole("button", { name: /Enterprise/ });
  expect(current).toHaveTextContent("Current plan");
  expect(current).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("button", { name: /Free/ })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "Finish setup" }));
  await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  expect(selectPlan).not.toHaveBeenCalled();
});

it("shows one plan price with a per-seat suffix only for seat pricing", async () => {
  renderBilling();
  expect(await screen.findByRole("button", { name: /Team/ })).toHaveTextContent("$30.00 per seat");
  expect(screen.getByRole("button", { name: /Pro/ })).toHaveTextContent("$20.00");
  expect(screen.queryByText(/base price/)).not.toBeInTheDocument();
});
