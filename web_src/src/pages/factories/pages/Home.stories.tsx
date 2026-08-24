import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { FactoryHomePage } from "./FactoryHomePage";

/**
 * Workspace Home page: the start point of a workspace. It offers one manual
 * task, or an optional ingestion automation. Mounted inside FactoriesLayout so
 * the sidebar chrome appears.
 */
const meta = {
  title: "Factories/Pages/Home",
  component: FactoryHomePage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoryHomePage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/home`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
