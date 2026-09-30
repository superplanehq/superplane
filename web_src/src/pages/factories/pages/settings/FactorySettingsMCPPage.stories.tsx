import type { Meta, StoryObj } from "@storybook/react-vite";
import type { FactoriesFactoryMcpClient } from "@/api-client";
import { FEATURE_SUPERPLANE_MCP_SERVER, FEATURE_WORKSPACE_MCP } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { CONFIGURED_MCP_RESOURCES } from "../../__fixtures__/agentResourceFixtures";
import {
  ARNOLD_USER,
  OPERATOR_USER,
  ORGANIZATION_USERS,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageIds";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

function mcpClientUserAvatarUrl(userId?: string) {
  if (!userId) {
    return undefined;
  }
  const member = ORGANIZATION_USERS.find((user) => user.id === userId);
  return member && "avatarUrl" in member ? member.avatarUrl : undefined;
}

const meta = {
  title: "Factories/Pages/Settings/MCP servers",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const mcpPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/mcp`;

const CONFIGURED_MCP_CLIENTS: FactoriesFactoryMcpClient[] = [
  {
    id: "mcp-client-cursor-leonardo",
    clientName: "Cursor",
    userId: STORYBOOK_ME_USER_ID,
    userName: STORYBOOK_ME_USER_NAME,
    userAvatarUrl: mcpClientUserAvatarUrl(STORYBOOK_ME_USER_ID),
    createdAt: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-cursor-arnold",
    clientName: "Cursor",
    userId: ARNOLD_USER.id,
    userName: ARNOLD_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(ARNOLD_USER.id),
    createdAt: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-cursor-alex",
    clientName: "Cursor",
    userId: REVIEWER_USER.id,
    userName: REVIEWER_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(REVIEWER_USER.id),
    createdAt: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-cursor-jamie",
    clientName: "Cursor",
    userId: OPERATOR_USER.id,
    userName: OPERATOR_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(OPERATOR_USER.id),
    createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-claude",
    clientName: "Claude Code",
    userId: STORYBOOK_ME_USER_ID,
    userName: STORYBOOK_ME_USER_NAME,
    userAvatarUrl: mcpClientUserAvatarUrl(STORYBOOK_ME_USER_ID),
    createdAt: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-vscode",
    clientName: "VS Code",
    userId: REVIEWER_USER.id,
    userName: REVIEWER_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(REVIEWER_USER.id),
    createdAt: new Date(Date.now() - 18 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-codex",
    clientName: "Codex",
    userId: ARNOLD_USER.id,
    userName: ARNOLD_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(ARNOLD_USER.id),
    createdAt: new Date(Date.now() - 4 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

function mcpHarness(fixture: FactoriesFixture) {
  return (
    <FactoriesHarness
      pathSuffix={mcpPath}
      experimentalFeatures={[FEATURE_WORKSPACE_MCP, FEATURE_SUPERPLANE_MCP_SERVER]}
      factoriesFixture={fixture}
    />
  );
}

/** No outbound or inbound connections. */
export const Empty: Story = {
  render: () =>
    mcpHarness({
      ...defaultFactoriesFixture,
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
      mcpClientsByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
    }),
};

/** Twelve outbound connections plus three inbound clients. */
export const Configured: Story = {
  render: () =>
    mcpHarness({
      ...defaultFactoriesFixture,
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: CONFIGURED_MCP_RESOURCES },
      mcpClientsByFactoryId: { [PRIMARY_FACTORY_ID]: CONFIGURED_MCP_CLIENTS },
    }),
};

/** Outbound and inbound list requests fail. */
export const Error: Story = {
  render: () =>
    mcpHarness({
      ...defaultFactoriesFixture,
      failAgentResourcesListForFactoryIds: [PRIMARY_FACTORY_ID],
      failMcpClientsListForFactoryIds: [PRIMARY_FACTORY_ID],
    }),
};
