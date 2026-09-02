import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { EMPTY_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { FactorySettingsLayout } from "../settings/FactorySettingsLayout";

/** Redesigned organization settings. Same chrome as workspace settings. */
const meta = {
  title: "Factories/Pages/Settings/Organization",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const organizationSettingsPath = (page: string) => `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/${page}`;

export const General: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("general")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Members: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("members")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Integrations: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("integrations")}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const ApiKeys: Story = {
  name: "API keys",
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("api-keys")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Secrets: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("secrets")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Spending: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("spending")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const SpendingEmpty: Story = {
  name: "Spending (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("spending")}
      factoriesFixture={{ ...defaultFactoriesFixture, organizationWorkspaceUsage: EMPTY_USAGE_REPORT }}
    />
  ),
};

export const SpendingBilling: Story = {
  name: "Spending (add hosted credit)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("spending")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationWorkspaceUsage: {
          ...EMPTY_USAGE_REPORT,
          remainingCreditCents: "0",
          grantTotalCents: "0",
          hostedBilledCents: "0",
          remainingCreditWarning: true,
          billingEnabled: true,
          hasBillingCustomer: false,
        },
        hostedCreditProducts: [
          { id: "prod-500", name: "Hosted credit 500", amountCents: "50000" },
          { id: "prod-25", name: "Hosted credit 25", amountCents: "2500" },
          { id: "prod-100", name: "Hosted credit 100", amountCents: "10000" },
        ],
      }}
    />
  ),
};

export const SpendingBillingInvoices: Story = {
  name: "Spending (manage invoices)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("spending")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationWorkspaceUsage: {
          ...EMPTY_USAGE_REPORT,
          remainingCreditCents: "14630",
          grantTotalCents: "15000",
          superplaneGrantCents: "5000",
          purchasedCreditCents: "10000",
          hostedBilledCents: "370",
          remainingCreditWarning: false,
          billingEnabled: true,
          hasBillingCustomer: true,
          invoices: [
            {
              id: "ord_100",
              createdAt: "2026-08-27T12:00:00Z",
              amountCents: "10000",
              status: "paid",
              productName: "$100 pack",
            },
          ],
        },
        hostedCreditProducts: [
          { id: "prod-500", name: "Hosted credit 500", amountCents: "50000" },
          { id: "prod-25", name: "Hosted credit 25", amountCents: "2500" },
          { id: "prod-100", name: "Hosted credit 100", amountCents: "10000" },
        ],
      }}
    />
  ),
};

export const SpendingCreditAdded: Story = {
  name: "Spending (credit added)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`${organizationSettingsPath("spending")}?credit=added`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationWorkspaceUsage: {
          ...EMPTY_USAGE_REPORT,
          remainingCreditCents: "2500",
          grantTotalCents: "2500",
          hostedBilledCents: "0",
          remainingCreditWarning: false,
          billingEnabled: true,
          hasBillingCustomer: true,
        },
        hostedCreditProducts: [{ id: "prod-25", name: "Hosted credit 25", amountCents: "2500" }],
      }}
    />
  ),
};
