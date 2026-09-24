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
    isLoading: false,
  }),
}));

describe("OrganizationSettingsBillingPage custom top-up", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

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

    const balance = await screen.findByTestId("billing-credit-balance");
    const topup = within(balance).getByTestId("billing-credit-topup");
    await waitFor(() => {
      expect(within(topup).getByRole("button", { name: "Top up" })).toBeEnabled();
    });
    await user.click(within(topup).getByRole("button", { name: "Top up" }));
    expect(await within(topup).findByRole("button", { name: "Custom" })).toBeEnabled();
    await user.click(within(topup).getByRole("button", { name: "Custom" }));

    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();
  }, 10000);
});
