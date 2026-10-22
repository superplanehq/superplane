import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  FEATURE_SUPERPLANE_MCP_SERVER,
  FEATURE_WORKSPACE_AGENT,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  CONFIGURED_MCP_RESOURCES,
  CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
  CONFIGURED_SKILLS,
  CONTEXT_REPO_RUNNER,
  CONTEXT_REPO_SUSS,
  SETUP_COMMANDS_RESOURCE,
} from "../../__fixtures__/agentResourceFixtures";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Agent",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;

const agentExperimentalFeatures = [FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS, FEATURE_SUPERPLANE_MCP_SERVER];

function agentHarness(fixture: FactoriesFixture) {
  return (
    <FactoriesHarness
      pathSuffix={agentPath}
      experimentalFeatures={agentExperimentalFeatures}
      factoriesFixture={fixture}
    />
  );
}

/** Agent page with no setup steps, context repos, MCP servers, or skills. */
export const Empty: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
    }),
};

/** Agent page with every section configured. */
export const Configured: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourceToolsById: CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
      agentResourcesByFactoryId: {
        [PRIMARY_FACTORY_ID]: [
          SETUP_COMMANDS_RESOURCE,
          CONTEXT_REPO_SUSS,
          CONTEXT_REPO_RUNNER,
          ...CONFIGURED_MCP_RESOURCES,
          ...CONFIGURED_SKILLS,
        ],
      },
    }),
};
