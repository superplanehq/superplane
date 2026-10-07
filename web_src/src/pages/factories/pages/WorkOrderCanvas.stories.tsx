import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { WorkOrderCanvas } from "./WorkOrderCanvas";

/**
 * Factory-only node chrome (WorkOrderCanvas): white cards, provider icons, tinted
 * status footers. Does not change non-factory CanvasPage / ComponentBase nodes.
 */
const meta = {
  title: "Factories/Pages/WorkOrderCanvas",
  component: WorkOrderCanvas,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrderCanvas>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ConfigurePhase: Story = {
  name: "Configure phase",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    const phase = line.steps?.[0]?.app?.app ?? "plan";
    return (
      <FactoriesHarness
        enableOnboarding={false}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}/phases/${phase}/configure`}
        factoriesFixture={defaultFactoriesFixture}
      />
    );
  },
};
