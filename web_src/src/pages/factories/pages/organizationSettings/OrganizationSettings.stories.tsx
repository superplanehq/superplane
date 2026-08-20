import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { OrganizationSettingsLayout } from "./OrganizationSettingsLayout";

const meta = {
  title: "Factories/Pages/OrganizationSettings",
  component: OrganizationSettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof OrganizationSettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const General: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/organization/general`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Workspaces: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/organization/workspaces`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
