import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";

/**
 * Lines page: card list → line detail with phase strip + per-step run board.
 */
const meta = {
  title: "Factories/Pages/Lines",
  component: LinesPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof LinesPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const linesListPath = `workspaces/${PRIMARY_FACTORY_ID}/lines`;

export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={linesListPath} factoriesFixture={defaultFactoriesFixture} />,
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
        pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/lines/${line.id}`}
        factoriesFixture={defaultFactoriesFixture}
      />
    );
  },
};
