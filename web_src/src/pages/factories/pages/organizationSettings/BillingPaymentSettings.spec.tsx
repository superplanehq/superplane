import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expect, it, vi } from "bun:test";
import { BillingPaymentSettings } from "./BillingPaymentSettings";

const selectPlan = vi.fn(async () => {
  throw new Error("Checkout unavailable");
});
vi.mock("@/api-client", () => ({
  organizationsSelectBillingPlan: selectPlan,
  organizationsCreatePaymentMethodSession: vi.fn(),
  organizationsCreateBillingPortalSession: vi.fn(),
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({
    data: {
      billingEnabled: true,
      hasPaymentMethod: false,
      requirePaymentMethod: true,
      availablePlans: [
        { key: "payg", name: "Pay-as-you-go", requirePaymentMethod: true, priceCents: "0" },
        { key: "pro", name: "Pro", requirePaymentMethod: true, subscriptionRequired: true, priceCents: "25000" },
      ],
    },
    refetch: vi.fn(),
  }),
}));

it("requires card setup for a free plan but lets paid checkout collect the card", async () => {
  const user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <BillingPaymentSettings organizationId="org-1" canManageBilling />
    </QueryClientProvider>,
  );
  expect(screen.getByRole("button", { name: "Select Pay-as-you-go" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Add payment method" })).toBeEnabled();
  await user.click(screen.getByRole("button", { name: "Subscribe to Pro" }));
  await waitFor(() =>
    expect(selectPlan).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1" },
        body: { planKey: "pro" },
      }),
    ),
  );
  expect(await screen.findByRole("alert")).toHaveTextContent("Checkout unavailable");
});
