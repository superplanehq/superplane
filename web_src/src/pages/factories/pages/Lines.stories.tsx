import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { refundLineCanvasFixture } from "../__fixtures__/factoryOwnedCanvasFixture";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";

/**
 * Lines page: card list → line detail with phase board.
 * Line stories pass a factory-owned canvas fixture so a card click opens
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
const lineCanvasFixture = refundLineCanvasFixture();

export const Populated: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={linesListPath}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={lineCanvasFixture}
    />
  ),
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
        factoriesFixture={defaultFactoriesFixture}
        appFixture={lineCanvasFixture}
      />
    );
  },
};
