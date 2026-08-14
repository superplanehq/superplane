import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { MissionsPage } from "./MissionsPage";
import { WikiPage } from "./WikiPage";

/**
 * Missions/Wiki landing pages — ComingSoon placeholders.
 */
const meta = {
  title: "Factories/Pages/Coming Soon",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Missions: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/missions`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Wiki: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/wiki`}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ wiki: WikiPage }}
    />
  ),
};

// Direct component references so Storybook's DevTools recognise them.
void MissionsPage;
void WikiPage;
