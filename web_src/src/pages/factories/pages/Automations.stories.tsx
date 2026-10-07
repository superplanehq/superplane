import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_APPS,
} from "../__fixtures__/factoryPageResponses";
import { emptyFactoriesFixture } from "../__fixtures__/factoryPageFixtureVariants";
import { AutomationsPage } from "./AutomationsPage";

/**
 * Automations page: v3-style card list + detail (automation card + Runs / Velocity tabs).
 */
const meta = {
  title: "Factories/Pages/Automations",
  component: AutomationsPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AutomationsPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const automationsPath = `workspaces/${PRIMARY_FACTORY_KEY}/automations`;
const plannerAppId = REFUND_FACTORY_APPS[0]?.id ?? "app-refund-planner";

export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={defaultFactoriesFixture} />,
};

export const DetailWithRuns: Story = {
  name: "Detail with runs",
  render: () => (
    <FactoriesHarness pathSuffix={`${automationsPath}/${plannerAppId}`} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const EmptyFactory: Story = {
  name: "Empty factory",
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={emptyFactoriesFixture} />,
};
