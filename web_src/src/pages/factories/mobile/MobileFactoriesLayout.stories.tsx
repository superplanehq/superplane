import type { Meta, StoryObj } from "@storybook/react-vite";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { MobileFactoriesLayout } from "./MobileFactoriesLayout";

const meta = {
  title: "Factories/Mobile/Workspace",
  component: MobileFactoriesLayout,
  parameters: { layout: "fullscreen" },
  globals: { viewport: { value: "mobile1", isRotated: false } },
} satisfies Meta<typeof MobileFactoriesLayout>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Board: Story = {
  render: () => (
    <FactoriesHarness
      enableOnboarding={false}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
    />
  ),
};
export const EmptyBoard: Story = {
  render: () => (
    <FactoriesHarness
      enableOnboarding={false}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        workOrdersByFactoryId: Object.fromEntries(defaultFactoriesFixture.factories.map((factory) => [factory.id, []])),
      }}
    />
  ),
};
export const More: Story = {
  render: () => (
    <FactoriesHarness
      enableOnboarding={false}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/more?lineId=${REFUND_LINE_PLAN_ID}`}
    />
  ),
};
