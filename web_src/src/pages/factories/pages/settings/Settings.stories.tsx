import type { Meta, StoryObj } from "@storybook/react-vite";

import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

/** Current factory settings chrome. Use the sidebar to open each page. */
const meta = {
  title: "Factories/Pages/Settings",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Current: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
      factoriesFixture={defaultFactoriesFixture}
      experimentalFeatures={[FEATURE_WORKSPACE_MODELS]}
    />
  ),
};
