import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { emptyFactoriesFixture, PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../__fixtures__/factoryPageResponses";
import { fiveStepLineFactoriesFixture, lineMetricsFactoriesFixture } from "../__fixtures__/lineMetricsFactoriesFixture";
import { LinesPage } from "./LinesPage";

/**
 * Line board is the workspace home: phase columns fill the pane. Cards open
 * the job-report popup (Storybook wireframe). The list story remains for
 * line management.
 */
const meta = {
  title: "Factories/Pages/Lines",
  component: LinesPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof LinesPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const linesListPath = `workspaces/${PRIMARY_FACTORY_KEY}/lines`;

export const Populated: Story = {
  name: "Line board",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};

export const LineList: Story = {
  name: "Line list",
  render: () => <FactoriesHarness pathSuffix={linesListPath} factoriesFixture={lineMetricsFactoriesFixture} />,
};

export const EmptyFactory: Story = {
  name: "Empty factory",
  render: () => <FactoriesHarness pathSuffix={linesListPath} factoriesFixture={emptyFactoriesFixture} />,
};

export const LineDetailFivePhases: Story = {
  name: "Line detail — five phases",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={fiveStepLineFactoriesFixture}
      />
    );
  },
};
