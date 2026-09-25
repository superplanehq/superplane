import type { Meta, StoryObj } from "@storybook/react-vite";

import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  HEADER_MCP_RESOURCE,
  INLINE_SKILL,
  MIXED_AGENT_RESOURCES,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NEEDS_RECONNECT_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
  OAUTH_VENDOR_REJECTED_RESOURCE,
  UI_UX_PRO_MAX_SKILL,
} from "../../__fixtures__/agentResourceFixtures";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Settings/MCP and skills",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const mcpPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/mcp`;
const skillsPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/skills`;

function withResources(resources: typeof MIXED_AGENT_RESOURCES, pathSuffix = mcpPath) {
  return (
    <FactoriesHarness
      pathSuffix={pathSuffix}
      experimentalFeatures={[FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS]}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        agentResourcesByFactoryId: {
          [PRIMARY_FACTORY_ID]: resources,
        },
      }}
    />
  );
}

export const MCPEmpty: Story = {
  render: () => withResources([]),
};

export const MCPCatalog: Story = {
  render: () => withResources([], `${mcpPath}?dialog=add`),
};

export const HeaderAuth: Story = {
  render: () => withResources([HEADER_MCP_RESOURCE]),
};

export const OAuthNotConnected: Story = {
  render: () => withResources([OAUTH_NOT_CONNECTED_RESOURCE]),
};

export const ConnectedGreenDot: Story = {
  render: () => withResources([OAUTH_CONNECTED_RESOURCE]),
};

export const OAuthNeedsReconnect: Story = {
  render: () => withResources([OAUTH_NEEDS_RECONNECT_RESOURCE]),
};

export const OAuthVendorRejected: Story = {
  render: () => withResources([OAUTH_VENDOR_REJECTED_RESOURCE]),
};

export const Mixed: Story = {
  render: () => withResources(MIXED_AGENT_RESOURCES),
};

export const SkillsEmpty: Story = {
  render: () => withResources([], skillsPath),
};

export const SkillsInline: Story = {
  render: () => withResources([INLINE_SKILL], skillsPath),
};

export const SkillsGitHub: Story = {
  render: () => withResources([UI_UX_PRO_MAX_SKILL], skillsPath),
};

export const SkillEditor: Story = {
  render: () => withResources([INLINE_SKILL], `${skillsPath}/new`),
};
