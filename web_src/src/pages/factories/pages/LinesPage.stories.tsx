import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";

/**
 * Storybook-only port of the v3 Lines page (list → stage board → canvas / configure).
 */
const meta = {
  title: "Factories/Pages/Lines",
  component: LinesPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof LinesPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const linesPath = `workspaces/${PRIMARY_FACTORY_ID}/lines`;

export const Default: Story = {
  render: () => <FactoriesHarness pathSuffix={linesPath} factoriesFixture={defaultFactoriesFixture} />,
};

export const Selected: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={`${linesPath}?line=bug-fixer`} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Configure: Story = {
  name: "Configure phase",
  render: () => (
    <FactoriesHarness
      pathSuffix={`${linesPath}/bug-fixer/phases/fix/configure`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
