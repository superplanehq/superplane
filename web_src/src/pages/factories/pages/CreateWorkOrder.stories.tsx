import type { Meta, StoryObj } from "@storybook/react-vite";

import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, EMPTY_FACTORY_KEY, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";

/**
 * Live create dialog. `/tasks/new` opens it over the list.
 * Task Refinement uses the request composer. Flag-off keeps the title form.
 */
const meta = {
  title: "Factories/Pages/Create Task",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const createPath = (factoryKey: string) => `workspaces/${factoryKey}/tasks/new`;

export const Empty: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={createPath(PRIMARY_FACTORY_KEY)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const RequestComposer: Story = {
  name: "Request composer",
  render: () => (
    <FactoriesHarness
      pathSuffix={createPath(PRIMARY_FACTORY_KEY)}
      factoriesFixture={defaultFactoriesFixture}
      experimentalFeatures={[FEATURE_FACTORY_CREATE_WITH_AGENT]}
    />
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
