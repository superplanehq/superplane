import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { BillingPaymentSettings } from "./BillingPaymentSettings";

const selectPlan = vi.fn();
const createSession = vi.fn();
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
  organizationsCreateBillingPortalSession: vi.fn(),
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({ data: billing, refetch }),
}));

beforeEach(() => {
  billing.hasPaymentMethod = false;
  billing.occupiedSeats = 2;
  billing.availablePlans![1].seatMaximum = 5;
  selectPlan.mockReset().mockResolvedValue({ data: {} });
  createSession.mockReset().mockResolvedValue({ data: { portalUrl: "https://polar.example/portal?token=test" } });
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
    const user = renderSettings();

    expect(screen.queryByRole("button", { name: "Add payment method" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Select Pay-as-you-go" }));
    await waitFor(() => expect(refetch).toHaveBeenCalled());
    expect(open).toHaveBeenCalledWith("about:blank", "_blank", expect.stringContaining("popup=yes"));
    expect(popup.location.replace).toHaveBeenCalledWith("https://polar.example/portal/settings?token=test");
    expect(selectPlan).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Select Pay-as-you-go" })).toBeDisabled();

    // Closing the provider window must still verify the card before deciding to activate.
    popup.closed = true;
    completeVerification({ data: { hasPaymentMethod: saved } });
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
  const user = renderSettings();
  await user.click(screen.getByRole("button", { name: "Subscribe to Pro" }));
  await waitFor(() => expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "pro" } })));
  expect(createSession).not.toHaveBeenCalled();
  expect(await screen.findByRole("alert")).toHaveTextContent("Checkout unavailable");
});

it("selects Pay-as-you-go directly when a card is already saved", async () => {
  billing.hasPaymentMethod = true;
  const open = vi.spyOn(window, "open");
  const user = renderSettings();
  await user.click(screen.getByRole("button", { name: "Select Pay-as-you-go" }));
  await waitFor(() => expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "payg" } })));
  expect(open).not.toHaveBeenCalled();
});

it("allows retry when the browser blocks card setup", async () => {
  vi.spyOn(window, "open").mockReturnValue(null);
  const user = renderSettings();
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
  const user = renderSettings();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(2);
  expect(seats).toHaveAttribute("min", "2");
  expect(seats).toHaveAttribute("max", "5");
  expect(selectPlan).not.toHaveBeenCalled();
  expect(screen.getByText("$20.00 per seat per month")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Increase Team seats" }));
  expect(seats).toHaveValue(3);
  expect(screen.getByText("Initial payment estimate: $60.00")).toBeInTheDocument();
  expect(screen.getByText("Monthly total: $60.00")).toBeInTheDocument();
  expect(screen.getByText("Estimates exclude tax.")).toBeInTheDocument();
  const subscribe = screen.getByRole("button", { name: "Subscribe to Team" });
  await user.dblClick(subscribe);
  await waitFor(() => expect(selectPlan).toHaveBeenCalledTimes(1));
  expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "team", seats: 3 } }));
  expect(subscribe).toBeDisabled();
  expect(seats).toBeDisabled();
  await act(async () => finishCheckout({ data: {} }));
  await waitFor(() => expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeEnabled());
});

it.each(["", "0", "1", "6", "2.5", "-1"])("blocks invalid Team seat input: %s", async (value) => {
  const user = renderSettings();
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
  const user = renderSettings();
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
  const user = renderSettings();
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(1);
  expect(seats).not.toHaveAttribute("max");
  await user.clear(seats);
  await user.type(seats, "51");
  expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeEnabled();
});

it("hides checkout controls from members without billing permission", () => {
  renderSettings(false);
  expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Subscribe to Team" })).not.toBeInTheDocument();
  expect(selectPlan).not.toHaveBeenCalled();
});

it.each(["missing usage", "maximum below occupancy"])(
  "blocks checkout when bounds cannot cover members: %s",
  (condition) => {
    if (condition === "missing usage") delete billing.occupiedSeats;
    else billing.availablePlans![1].seatMaximum = 1;
    renderSettings();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Subscribe to Team" })).toBeDisabled();
    expect(selectPlan).not.toHaveBeenCalled();
  },
);
