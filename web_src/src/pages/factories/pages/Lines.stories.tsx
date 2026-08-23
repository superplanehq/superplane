import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../__fixtures__/factoryPageResponses";
import { emptyFactoriesFixture } from "../__fixtures__/factoryPageFixtureVariants";
import { fiveStepLineFactoriesFixture, lineMetricsFactoriesFixture } from "../__fixtures__/lineMetricsFactoriesFixture";
import { LinesPage } from "./LinesPage";

/**
 * Line board is the workspace home: phase columns fill the pane. Cards open
 * the terminal-log run popup. The list story remains for line management.
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

export const LineBoardIntake: Story = {
  name: "Line board — intake",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};

export const LineBoardIntakeAnalyzing: Story = {
  name: "Line board — intake tree",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&source=github-issues`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
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
