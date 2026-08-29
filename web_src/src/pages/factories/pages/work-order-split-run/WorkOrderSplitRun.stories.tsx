import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { REFUND_IMPLEMENTER_APP } from "../../__fixtures__/factoryOwnedCanvasFixture";
import {
  LINE_RUN_IMPLEMENT_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";

/**
 * Line board with the work-order popup. Open a card: finished steps stay
 * collapsed and the current step expands into a log.
 */
const meta = {
  title: "Factories/Pages/Task Split Run",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Running: Story = {
  name: "Terminal log",
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

export const AutomationRun: Story = {
  name: "Automation run",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run?from=lines&lineId=${line.id}&run=${LINE_RUN_IMPLEMENT_ID}&orderNumber=103&canvas=implementation`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};
