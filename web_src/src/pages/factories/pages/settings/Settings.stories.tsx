import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

/**
 * Factory settings — dedicated layout with its own left nav and General page content.
 */
const meta = {
  title: "Factories/Pages/Settings",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const General: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/settings/general`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const RepositoriesSoon: Story = {
  name: "Repositories (Soon)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/settings/repositories`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const MembersSoon: Story = {
  name: "Members (Soon)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/settings/members`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
