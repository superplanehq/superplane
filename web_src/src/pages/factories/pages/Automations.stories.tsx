import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  PRIMARY_FACTORY_ID,
} from "../__fixtures__/factoryPageResponses";
import { AutomationsPage } from "./AutomationsPage";

/**
 * Automations page: master–detail Lines + Phases with an Apps section on the right.
 */
const meta = {
  title: "Factories/Pages/Automations",
  component: AutomationsPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AutomationsPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const automationsPath = `workspaces/${PRIMARY_FACTORY_ID}/automations`;

export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={defaultFactoriesFixture} />,
};

export const EmptyFactoryLines: Story = {
  name: "Empty factory",
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={emptyFactoriesFixture} />,
};
