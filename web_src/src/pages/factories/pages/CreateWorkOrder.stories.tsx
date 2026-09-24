import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  EMPTY_FACTORY_KEY,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";

/**
 * Live create dialog. `/tasks/new` opens it over the list.
 * Planning on uses the request composer. Planning off keeps the title form.
 */
const meta = {
  title: "Factories/Pages/Create Task",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const createPath = (factoryKey: string) => `workspaces/${factoryKey}/tasks/new`;

const planningOffFixture = {
  ...defaultFactoriesFixture,
  factories: defaultFactoriesFixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID
      ? { ...factory, planning: { enabled: false, clarity: true, confidence: true } }
      : factory,
  ),
};

export const Empty: Story = {
  render: () => <FactoriesHarness pathSuffix={createPath(PRIMARY_FACTORY_KEY)} factoriesFixture={planningOffFixture} />,
};

export const RequestComposer: Story = {
  name: "Request composer",
  render: () => (
    <FactoriesHarness pathSuffix={createPath(PRIMARY_FACTORY_KEY)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Assigned: Story = {
  name: "Assignees available",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const NoLines: Story = {
  name: "No lines",
  render: () => (
    <FactoriesHarness pathSuffix={createPath(EMPTY_FACTORY_KEY)} factoriesFixture={defaultFactoriesFixture} />
  ),
};
