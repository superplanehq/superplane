import type { Meta, StoryObj } from "@storybook/react-vite";

import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { EMPTY_ORG_SPENDING_REPORT } from "../../__fixtures__/spendingReportFixtures";
import { SPENT_CREDIT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
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
        organizationCreditGrants: MIXED_CREDIT_GRANTS,
      }}
    />
  ),
};

export const BillingEmptyHistory: Story = {
  name: "Billing (empty history)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationCreditGrants: [],
      }}
    />
  ),
};

export const BillingEmptyCredit: Story = {
  name: "Billing (empty credit)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
      }}
    />
  ),
};
