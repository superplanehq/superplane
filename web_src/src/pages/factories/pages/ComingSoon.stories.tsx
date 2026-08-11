import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";
import { MissionsPage } from "./MissionsPage";
import { VelocityPage } from "./VelocityPage";
import { WikiPage } from "./WikiPage";

/**
 * Missions/Lines/Wiki/Velocity landing pages — all render the ComingSoon placeholder.
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
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/missions`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Lines: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/lines`} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Wiki: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/wiki`} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Velocity: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/velocity`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

// Direct component references so Storybook's DevTools recognise them.
void LinesPage;
void MissionsPage;
void VelocityPage;
void WikiPage;
