import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture } from "../../__fixtures__/factoryPageResponses";
import { EMPTY_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { OrganizationSettingsLayout } from "./OrganizationSettingsLayout";

const meta = {
  title: "Factories/Pages/OrganizationSettings",
  component: OrganizationSettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof OrganizationSettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const General: Story = {
  render: () => <FactoriesHarness pathSuffix="organization/general" factoriesFixture={defaultFactoriesFixture} />,
};

export const Workspaces: Story = {
  render: () => <FactoriesHarness pathSuffix="organization/workspaces" factoriesFixture={defaultFactoriesFixture} />,
};

export const WorkspaceUsage: Story = {
  name: "Workspace usage",
  render: () => (
    <FactoriesHarness pathSuffix="organization/workspace-usage" factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const WorkspaceUsageEmpty: Story = {
  name: "Workspace usage (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix="organization/workspace-usage"
      factoriesFixture={{ ...defaultFactoriesFixture, organizationWorkspaceUsage: EMPTY_USAGE_REPORT }}
    />
  ),
};

export const WorkspaceUsageBilling: Story = {
  name: "Workspace usage (add hosted credit)",
  render: () => (
    <FactoriesHarness
      pathSuffix="organization/workspace-usage"
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

export const WorkspaceUsageBillingInvoices: Story = {
  name: "Workspace usage (manage invoices)",
  render: () => (
    <FactoriesHarness
      pathSuffix="organization/workspace-usage"
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

export const WorkspaceUsageCreditAdded: Story = {
  name: "Workspace usage (credit added)",
  render: () => (
    <FactoriesHarness
      pathSuffix="organization/workspace-usage?credit=added"
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

export const Integrations: Story = {
  render: () => <FactoriesHarness pathSuffix="organization/integrations" factoriesFixture={defaultFactoriesFixture} />,
};
