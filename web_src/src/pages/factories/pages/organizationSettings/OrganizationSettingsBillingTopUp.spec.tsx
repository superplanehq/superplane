import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  BUSINESS_ORGANIZATION_BILLING,
  PURCHASED_CREDIT_USAGE_REPORT,
  STORYBOOK_CUSTOM_CREDIT_PRODUCT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: () => true,
    hasPermission: () => true,
    isLoading: false,
  }),
}));

describe("OrganizationSettingsBillingPage top-up", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens Polar checkout for a single hosted credit pack", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[0]],
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            billingEnabled: true,
            hasBillingCustomer: false,
          },
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    await waitFor(() => {
      expect(within(balance).getByRole("button", { name: "Top up" })).toBeEnabled();
    });
    expect(within(balance).queryByRole("button", { name: "Buy more" })).not.toBeInTheDocument();
    expect(within(balance).queryByRole("button", { name: "$50" })).not.toBeInTheDocument();
    await user.click(within(balance).getByRole("button", { name: "Top up" }));
    const dialog = await screen.findByRole("dialog", { name: "Add hosted credit" });
    await user.click(within(dialog).getByRole("button", { name: "$50" }));
    const confirm = within(dialog).getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    expect(dialog).toHaveTextContent("$3.16");
    expect(dialog).toHaveTextContent("$53.16");
    expect(assign).not.toHaveBeenCalled();
    await user.click(confirm);

    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook"));
    vi.unstubAllGlobals();
  }, 10000);

  it("opens Polar checkout for a custom hosted credit pack", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [...STORYBOOK_HOSTED_CREDIT_PRODUCTS, STORYBOOK_CUSTOM_CREDIT_PRODUCT],
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            billingEnabled: true,
            hasBillingCustomer: true,
          },
        }}
      />,
    );

    const checkoutBodies: unknown[] = [];
    const captureCheckout = async (request: Request) => {
      if (new URL(request.url).pathname.endsWith("/hosted-credit-checkout")) {
        checkoutBodies.push(await request.clone().json());
      }
      return request;
    };
    client.interceptors.request.use(captureCheckout);
    const balance = await screen.findByTestId("billing-credit-balance");
    await waitFor(() => {
      expect(within(balance).getByRole("button", { name: "Top up" })).toBeEnabled();
    });
    await user.click(within(balance).getByRole("button", { name: "Top up" }));
    const dialog = await screen.findByRole("dialog", { name: "Add hosted credit" });
    await user.click(within(dialog).getByRole("button", { name: "Custom" }));
    await user.type(within(dialog).getByRole("textbox", { name: "Credit amount in USD" }), "100");
    const confirm = within(dialog).getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    expect(dialog).toHaveTextContent("$100.00");
    expect(dialog).toHaveTextContent("$5.79");
    expect(dialog).toHaveTextContent("$105.79");
    expect(assign).not.toHaveBeenCalled();
    await user.click(confirm);

    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook"));
    expect(checkoutBodies).toEqual([
      { productId: "prod-custom", creditAmountCents: "10000", expectedSubtotalCents: "10579" },
    ]);
    client.interceptors.request.eject(captureCheckout);
    vi.unstubAllGlobals();
  }, 10000);
});
