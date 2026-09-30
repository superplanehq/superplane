import type { Meta, StoryObj } from "@storybook/react-vite";
import { FEATURE_SUPERPLANE_MCP_SERVER, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  CONFIGURED_MCP_RESOURCES,
  CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
  CONFIGURED_SKILLS,
} from "../../__fixtures__/agentResourceFixtures";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Settings/Agent",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent`;

const agentExperimentalFeatures = [
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
  FEATURE_SUPERPLANE_MCP_SERVER,
];

function agentHarness(fixture: FactoriesFixture) {
  return (
    <FactoriesHarness
      pathSuffix={agentPath}
      experimentalFeatures={agentExperimentalFeatures}
      factoriesFixture={fixture}
    />
  );
}

/** Agent page with no outbound MCP servers or skills. */
export const Empty: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
    }),
};

/** Outbound MCP servers and skills configured. */
export const Configured: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourceToolsById: CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
      agentResourcesByFactoryId: {
        [PRIMARY_FACTORY_ID]: [...CONFIGURED_MCP_RESOURCES, ...CONFIGURED_SKILLS],
      },
    }),
};

/** Outbound MCP list request fails. */
export const Error: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      failAgentResourcesListForFactoryIds: [PRIMARY_FACTORY_ID],
    }),
};
