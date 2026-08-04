import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoryDetailPage } from "./FactoryDetailPage";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyWorkOrdersFactoriesFixture,
  PRIMARY_FACTORY_ID,
} from "./__fixtures__/factoryPageResponses";

/**
 * Factory detail view: header + Work Orders panel (filters, cards, dispatch)
 * with the Apps and Lines sidebars beside it. Uses the shared factories
 * fixture so navigation into a work order or line editor stays live.
 */
const meta = {
  title: "Pages/Factories/Detail",
  component: FactoryDetailPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof FactoryDetailPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const detailPath = `factories/${PRIMARY_FACTORY_ID}`;

/**
 * Populated detail view. Lands on the page's default `mine + open` filters,
 * so the list shows the two open orders assigned to `storybook-user`. Flip
 * status filters to reveal running / failed / closed variants, and the
 * owner filter to see orders belonging to teammates.
 */
export const Populated: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={detailPath} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Only closed history remains — the default `mine + open` filters land on the "no matches" state. */
export const EmptyWorkOrders: Story = {
  name: "Empty Work Orders",
  render: () => (
    <FactoriesHarness pathSuffix={detailPath} factoriesFixture={emptyWorkOrdersFactoriesFixture} />
  ),
};
