import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, emptyFactoriesFixture } from "../__fixtures__/factoryPageResponses";
import { FactoriesIndexPage } from "../FactoriesIndexPage";

/**
 * Landing route `/:orgId/workspaces`: redirects to the last-visited (or first)
 * factory when any exist, and shows the create-workspace empty state otherwise.
 */
const meta = {
  title: "Factories/Pages/Index",
  component: FactoriesIndexPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoriesIndexPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Populated org — auto-redirects to the primary factory's overview. */
export const RedirectsToPrimary: Story = {
  name: "Redirect to primary",
  render: () => <FactoriesHarness pathSuffix="workspaces" factoriesFixture={defaultFactoriesFixture} />,
};

/** No factories yet — shows the "Create workspace" empty state. */
export const EmptyState: Story = {
  name: "Empty state",
  render: () => <FactoriesHarness pathSuffix="workspaces" factoriesFixture={emptyFactoriesFixture} />,
};
