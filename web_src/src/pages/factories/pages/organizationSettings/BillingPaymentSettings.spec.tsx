import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { BillingPaymentSettings } from "./BillingPaymentSettings";

const selectPlan = vi.fn();
const createSession = vi.fn();
const createPortalSession = vi.fn();
const refetch = vi.fn();
const billing: OrganizationsDescribeOrganizationBillingResponse = {
  billingEnabled: true,
  occupiedSeats: 2,
  hasPaymentMethod: false,
  requirePaymentMethod: false,
  availablePlans: [
    { key: "payg", name: "Pay-as-you-go", requirePaymentMethod: true, priceCents: "0" },
    { key: "team", name: "Team", perSeat: true, subscriptionRequired: true, priceCents: "2000", seatMaximum: 5 },
    { key: "pro", name: "Pro", requirePaymentMethod: true, subscriptionRequired: true, priceCents: "25000" },
  ],
};
vi.mock("@/api-client", () => ({
  organizationsSelectBillingPlan: selectPlan,
  organizationsCreatePaymentMethodSession: createSession,
  organizationsCreateBillingPortalSession: createPortalSession,
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({ data: billing, refetch }),
}));

beforeEach(() => {
  delete billing.planSource;
  delete billing.creditPurchaseAllowed;
  delete billing.catalogKey;
  delete billing.polarSubscriptionStatus;
  delete billing.paymentFailedAt;
  delete billing.paidAccessEndsAt;
  billing.hasPaymentMethod = false;
  billing.occupiedSeats = 2;
  billing.pendingInvitationSeats = 0;
  billing.availablePlans![1].seatMaximum = 5;
  selectPlan.mockReset().mockResolvedValue({ data: {} });
  createSession.mockReset().mockResolvedValue({ data: { portalUrl: "https://polar.example/portal?token=test" } });
  createPortalSession.mockReset().mockResolvedValue({ data: { portalUrl: "https://polar.example/portal" } });
  refetch.mockReset().mockResolvedValue({ data: billing });
});
afterEach(() => vi.restoreAllMocks());

function renderSettings(canManageBilling = true) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <BillingPaymentSettings organizationId="org-1" canManageBilling={canManageBilling} />
    </QueryClientProvider>,
  );
  return userEvent.setup();
}

async function renderPlanChoices() {
  const user = renderSettings();
  await user.click(screen.getByRole("button", { name: "Change plan" }));
  return user;
}

async function renderTeamCheckout() {
  const user = await renderPlanChoices();
  await user.click(screen.getByRole("button", { name: "Review Team seats" }));
  return user;
}

it.each([true, false])(
  "only activates Pay-as-you-go when card setup verifies a saved card (%s)",
  async (saved) => {
    let completeVerification!: (result: { data: { hasPaymentMethod: boolean } }) => void;
    refetch.mockImplementation(
      () =>
        new Promise((resolve) => {
          completeVerification = resolve;
        }),
    );
    const popup = { closed: false, close: vi.fn(), location: { replace: vi.fn() }, opener: window };
    const open = vi.spyOn(window, "open").mockReturnValue(popup as unknown as Window);
    const user = await renderPlanChoices();

    await user.click(screen.getByRole("button", { name: "Select Pay-as-you-go" }));
    await waitFor(() => expect(refetch).toHaveBeenCalled());
    expect(open).toHaveBeenCalledWith("about:blank", "_blank", expect.stringContaining("popup=yes"));
    expect(popup.location.replace).toHaveBeenCalledWith("https://polar.example/portal/settings?token=test");
    expect(selectPlan).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Select Pay-as-you-go" })).toBeDisabled();

    // Closing the provider window must still verify the card before deciding to activate.
    popup.closed = true;
    await act(async () => completeVerification({ data: { hasPaymentMethod: saved } }));
    await waitFor(() => expect(screen.queryByRole("status")).not.toBeInTheDocument());
    if (saved) {
      await waitFor(() =>
        expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "payg" } })),
      );
      expect(popup.close).toHaveBeenCalled();
    } else {
      expect(selectPlan).not.toHaveBeenCalled();
      expect(screen.getByRole("button", { name: "Select Pay-as-you-go" })).toBeEnabled();
    }
  },
  15000,
);

it("lets paid checkout collect the card and shows activation errors", async () => {
  selectPlan.mockRejectedValue(new Error("Checkout unavailable"));
  const user = await renderPlanChoices();
  await user.click(screen.getByRole("button", { name: "Subscribe to Pro" }));
  await waitFor(() => expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "pro" } })));
  expect(createSession).not.toHaveBeenCalled();
  expect(await screen.findByRole("alert")).toHaveTextContent("Checkout unavailable");
});

it("selects Pay-as-you-go directly when a card is already saved", async () => {
  billing.hasPaymentMethod = true;
  const open = vi.spyOn(window, "open");
  const user = await renderPlanChoices();
  await user.click(screen.getByRole("button", { name: "Select Pay-as-you-go" }));
  await waitFor(() => expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "payg" } })));
  expect(open).not.toHaveBeenCalled();
});

it("allows retry when the browser blocks card setup", async () => {
  vi.spyOn(window, "open").mockReturnValue(null);
  const user = await renderPlanChoices();
  await user.click(screen.getByRole("button", { name: "Select Pay-as-you-go" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Allow pop-ups");
  expect(selectPlan).not.toHaveBeenCalled();
  expect(screen.getByRole("button", { name: "Select Pay-as-you-go" })).toBeEnabled();
});

it("reviews Team seats and estimates before submitting the fixed count", async () => {
  let finishCheckout!: (result: { data: object }) => void;
  selectPlan.mockImplementation(
    () =>
      new Promise((resolve) => {
        finishCheckout = resolve;
      }),
  );
  const user = await renderTeamCheckout();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(2);
  expect(seats).toHaveAttribute("min", "2");
  expect(seats).toHaveAttribute("max", "5");
  expect(selectPlan).not.toHaveBeenCalled();
  expect(screen.getByText("$20.00 per seat per month")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Increase Team seats" }));
  expect(seats).toHaveValue(3);
  expect(screen.getByText("$60.00")).toBeInTheDocument();
  expect(screen.getByText("Estimates exclude tax.")).toBeInTheDocument();
  const subscribe = screen.getByRole("button", { name: "Subscribe to Team" });
  await user.dblClick(subscribe);
  await waitFor(() => expect(selectPlan).toHaveBeenCalledTimes(1));
  expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "team", seats: 3 } }));
  expect(subscribe).toBeDisabled();
  expect(seats).toBeDisabled();
  await act(async () => finishCheckout({ data: {} }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await user.click(screen.getByRole("button", { name: "Change plan" }));
  expect(screen.getByRole("button", { name: "Review Team seats" })).toBeEnabled();
});

it.each(["", "0", "1", "6", "2.5", "-1"])("blocks invalid Team seat input: %s", async (value) => {
  const user = await renderTeamCheckout();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  await user.clear(seats);
  if (value) await user.type(seats, value);
  expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeDisabled();
  expect(selectPlan).not.toHaveBeenCalled();
});

it("refreshes bounds after an error and requires another explicit submission", async () => {
  refetch.mockImplementation(async () => {
    billing.occupiedSeats = 4;
    return { data: billing };
  });
  selectPlan.mockRejectedValueOnce(new Error("Select at least 4 seats to cover current members."));
  const user = await renderTeamCheckout();
  await user.click(screen.getByRole("button", { name: "Subscribe to Team" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Select at least 4 seats");
  await waitFor(() => expect(refetch).toHaveBeenCalledTimes(1));
  expect(screen.getByRole("spinbutton", { name: "Team seats" })).toHaveAttribute("min", "4");
  expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeDisabled();
  expect(selectPlan).toHaveBeenCalledTimes(1);
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  await user.clear(seats);
  await user.type(seats, "4");
  await user.click(screen.getByRole("button", { name: "Subscribe to Team" }));
  await waitFor(() => expect(selectPlan).toHaveBeenCalledTimes(2));
});

it("supports an unlimited maximum and a zero-member minimum of one", async () => {
  billing.occupiedSeats = 0;
  delete billing.availablePlans![1].seatMaximum;
  const user = await renderTeamCheckout();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(1);
  expect(seats).not.toHaveAttribute("max");
  await user.clear(seats);
  await user.type(seats, "51");
  expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeEnabled();
});

it("defaults Team seats to members plus pending invitations", async () => {
  billing.occupiedSeats = 1;
  billing.pendingInvitationSeats = 2;
  const user = await renderTeamCheckout();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(3);
  expect(seats).toHaveAttribute("min", "1");
  expect(
    screen.getByText("The default includes 2 seats for pending invitations. You can remove these seats."),
  ).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Subscribe to Team" }));
  await waitFor(() =>
    expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "team", seats: 3 } })),
  );
});

it.each([0, undefined])("defaults Team seats to members with no pending invitations: %s", async (pending) => {
  billing.pendingInvitationSeats = pending;
  await renderTeamCheckout();
  expect(screen.getByRole("spinbutton", { name: "Team seats" })).toHaveValue(2);
  expect(screen.queryByText(/pending invitation/)).not.toBeInTheDocument();
});

it("caps the pending default at the plan maximum and lets the admin pay for members only", async () => {
  billing.pendingInvitationSeats = 5;
  const user = await renderTeamCheckout();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(5);
  expect(screen.getByRole("status")).toHaveTextContent(
    "The plan maximum is 5 seats. 2 pending invitations cannot get a seat.",
  );
  const subscribe = screen.getByRole("button", { name: "Subscribe to Team" });
  expect(subscribe).toBeEnabled();
  await user.clear(seats);
  await user.type(seats, "2");
  expect(subscribe).toBeEnabled();
  await user.click(subscribe);
  await waitFor(() =>
    expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "team", seats: 2 } })),
  );
});

it("hides checkout controls from members without billing permission", () => {
  renderSettings(false);
  expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Subscribe to Team" })).not.toBeInTheDocument();
  expect(selectPlan).not.toHaveBeenCalled();
});

it.each(["missing usage", "maximum below occupancy"])(
  "blocks checkout when bounds cannot cover members: %s",
  async (condition) => {
    billing.pendingInvitationSeats = 3;
    if (condition === "missing usage") delete billing.occupiedSeats;
    else billing.availablePlans![1].seatMaximum = 1;
    await renderTeamCheckout();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeDisabled();
    expect(selectPlan).not.toHaveBeenCalled();
  },
);

const FAILED_AT = "2026-10-20T10:00:00Z";
const ACCESS_ENDS_AT = "2026-10-27T10:00:00Z";
const failedOn = new Date(FAILED_AT).toLocaleDateString();
const accessEndsOn = new Date(ACCESS_ENDS_AT).toLocaleDateString();

it("shows the failure and the date Team features end while a grace period is open", () => {
  Object.assign(billing, {
    polarSubscriptionStatus: "past_due",
    paymentFailedAt: FAILED_AT,
    paidAccessEndsAt: ACCESS_ENDS_AT,
  });
  renderSettings();
  expect(screen.getByRole("alert")).toHaveTextContent(
    `The renewal payment failed on ${failedOn}. Update the payment method by ${accessEndsOn} to keep Team features.`,
  );
});

it("says Team features are off when no grace period applies", () => {
  Object.assign(billing, { polarSubscriptionStatus: "past_due", paymentFailedAt: FAILED_AT });
  renderSettings();
  expect(screen.getByRole("alert")).toHaveTextContent(
    `The renewal payment failed on ${failedOn}. Team features are off until you update the payment method.`,
  );
});

it("opens the Polar payment method page from the banner", async () => {
  Object.assign(billing, { polarSubscriptionStatus: "past_due", paymentFailedAt: FAILED_AT });
  const assign = vi.fn();
  const previousAssign = window.location.assign.bind(window.location);
  window.location.assign = assign;
  try {
    const user = renderSettings();
    await user.click(screen.getByRole("button", { name: "Update payment method" }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.example/portal/settings?token=test"));
    expect(createSession).toHaveBeenCalledTimes(1);
    expect(selectPlan).not.toHaveBeenCalled();
  } finally {
    window.location.assign = previousAssign;
  }
});

it("shows a failed portal call on the banner and lets the owner retry", async () => {
  Object.assign(billing, { polarSubscriptionStatus: "past_due", paymentFailedAt: FAILED_AT });
  createSession.mockRejectedValueOnce(new Error("Polar is unavailable"));
  const user = renderSettings();
  await user.click(screen.getByRole("button", { name: "Update payment method" }));
  await waitFor(() => expect(screen.getAllByRole("alert").at(-1)).toHaveTextContent("Polar is unavailable"));
  expect(screen.getByRole("button", { name: "Update payment method" })).toBeEnabled();
});

it.each(["past_due", "paused"])("hides plan checkout while the subscription is %s", (status) => {
  Object.assign(billing, { polarSubscriptionStatus: status, paymentFailedAt: FAILED_AT });
  renderSettings();
  expect(screen.queryByRole("spinbutton", { name: "Team seats" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /^Subscribe to / })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /^Select / })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Add payment method" })).not.toBeInTheDocument();
});

it("keeps the banner text but removes the action for members without billing permission", () => {
  Object.assign(billing, { polarSubscriptionStatus: "past_due", paymentFailedAt: FAILED_AT });
  renderSettings(false);
  expect(screen.getByRole("alert")).toHaveTextContent("The renewal payment failed on");
  expect(screen.queryByRole("button", { name: "Update payment method" })).not.toBeInTheDocument();
});

it.each([false, true])("opens the portal for a paused subscription with a saved card: %s", async (saved) => {
  Object.assign(billing, { polarSubscriptionStatus: "paused", hasPaymentMethod: saved });
  const assign = vi.fn();
  const previousAssign = window.location.assign.bind(window.location);
  window.location.assign = assign;
  try {
    const user = renderSettings();
    await user.click(screen.getByRole("button", { name: "Manage payment method" }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.example/portal"));
    expect(createPortalSession).toHaveBeenCalledWith(expect.objectContaining({ path: { id: "org-1" }, body: {} }));
    expect(createSession).not.toHaveBeenCalled();
    expect(selectPlan).not.toHaveBeenCalled();
  } finally {
    window.location.assign = previousAssign;
  }
});

it("opens plan comparison before Team seat review without starting checkout", async () => {
  const user = renderSettings();
  expect(screen.queryByRole("button", { name: "Subscribe to Pro" })).not.toBeInTheDocument();
  expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Change plan" }));
  expect(screen.getByRole("dialog", { name: "Change plan" })).toBeInTheDocument();
  expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Review Team seats" }));
  expect(screen.getByRole("spinbutton", { name: "Team seats" })).toHaveValue(2);
  expect(selectPlan).not.toHaveBeenCalled();
  await user.click(screen.getByRole("button", { name: "Back to plans" }));
  expect(screen.getByRole("button", { name: "Subscribe to Pro" })).toBeEnabled();
  expect(selectPlan).not.toHaveBeenCalled();
  await user.click(screen.getByRole("button", { name: "Close" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "Change plan" })).toHaveFocus());
});

it("manages an existing paid subscription in the portal without creating another checkout", async () => {
  billing.planSource = "polar";
  billing.creditPurchaseAllowed = true;
  billing.catalogKey = "team";
  const assign = vi.fn();
  const previousAssign = window.location.assign.bind(window.location);
  window.location.assign = assign;
  try {
    const user = await renderPlanChoices();
    expect(screen.queryByRole("button", { name: "Subscribe to Pro" })).not.toBeInTheDocument();
    expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Manage subscription" }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.example/portal"));
    expect(selectPlan).not.toHaveBeenCalled();
  } finally {
    window.location.assign = previousAssign;
  }
});
