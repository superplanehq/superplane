import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { EMPTY_ORG_SPENDING_REPORT } from "../../__fixtures__/spendingReportFixtures";
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
