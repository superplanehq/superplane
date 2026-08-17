import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  emptyFactoriesFixture,
  fiveStepLineFactoriesFixture,
  lineMetricsFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";

/**
 * Lines page: metric cards on the list, phase board on line detail.
 * FactoriesHarness supplies a factory-owned canvas so a run-card click opens
 * the automation run instead of Overview.
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
  render: () => <FactoriesHarness pathSuffix={linesListPath} factoriesFixture={lineMetricsFactoriesFixture} />,
};

export const EmptyFactory: Story = {
  name: "Empty factory",
  render: () => <FactoriesHarness pathSuffix={linesListPath} factoriesFixture={emptyFactoriesFixture} />,
};

export const LineDetail: Story = {
  name: "Line detail",
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
