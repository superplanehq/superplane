import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { BillingPaymentSettings } from "./BillingPaymentSettings";

const selectPlan = vi.fn();
const createSession = vi.fn();
const refetch = vi.fn();
const billing = {
  billingEnabled: true,
  hasPaymentMethod: false,
  requirePaymentMethod: false,
  availablePlans: [
    { key: "payg", name: "Pay-as-you-go", requirePaymentMethod: true, priceCents: "0" },
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
  selectPlan.mockReset().mockResolvedValue({ data: {} });
  createSession.mockReset().mockResolvedValue({ data: { portalUrl: "https://polar.example/portal?token=test" } });
  refetch.mockReset();
});
afterEach(() => vi.restoreAllMocks());

function renderSettings() {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <BillingPaymentSettings organizationId="org-1" canManageBilling />
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
