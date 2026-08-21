import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";

/**
 * Line board with the terminal-log run popup. Open a card: finished steps
 * stay collapsed, the current step expands, and a log step click selects
 * that step on the canvas. Expand opens the full automation run page.
 *
 * This story replaces the factory app canvas run view.
 */
const meta = {
  title: "Factories/Pages/Work Order Split Run",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Running: Story = {
  name: "Terminal log and canvas",
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
