import type { Meta, StoryObj } from "@storybook/react-vite";

import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { EMPTY_ORG_SPENDING_REPORT } from "../../__fixtures__/spendingReportFixtures";
import {
  DEFAULT_FACTORY_USAGE,
  PURCHASED_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";
import { FactorySettingsLayout } from "../settings/FactorySettingsLayout";

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
      factoriesFixture={{ ...defaultFactoriesFixture, organizationSpendingReport: EMPTY_ORG_SPENDING_REPORT }}
    />
  ),
};

export const Billing: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationCreditGrants: MIXED_CREDIT_GRANTS,
        organizationWorkspaceUsage: {
          ...PURCHASED_CREDIT_USAGE_REPORT,
          billingEnabled: true,
          hasBillingCustomer: true,
          invoices: [
            {
              id: "ord_storybook_1",
              createdAt: "2026-09-01T12:00:00Z",
              amountCents: "2500",
              status: "paid",
              productName: "Hosted credit 25",
            },
          ],
        },
      }}
    />
  ),
};

export const BillingEmptyHistory: Story = {
  name: "Billing (trial welcome)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationCreditGrants: [],
        organizationWorkspaceUsage: {
          ...defaultFactoriesFixture.organizationWorkspaceUsage!,
          billingEnabled: true,
          hasBillingCustomer: false,
        },
      }}
    />
  ),
};

export const BillingEmptyNoCard: Story = {
  name: "Billing (empty credit, no card)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationWorkspaceUsage: {
          ...DEFAULT_FACTORY_USAGE,
          remainingCreditCents: "0",
          hostedBilledCents: "5000",
          remainingCreditWarning: true,
          billingEnabled: true,
          hasBillingCustomer: false,
        },
      }}
    />
  ),
};

export const BillingEmptyCredit: Story = {
  name: "Billing (empty credit, card on file)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
      }}
    />
  ),
};
