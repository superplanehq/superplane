import type { Meta, StoryObj } from "@storybook/react-vite";
import type { FactoriesFactoryMcpClient } from "@/api-client";
import { FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
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

const meta = {
  title: "Factories/Pages/Settings/MCP Server",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const superplaneMcpServerPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/superplane-mcp-server`;

function mcpClientUserAvatarUrl(userId?: string) {
  if (!userId) {
    return undefined;
  }
  const member = ORGANIZATION_USERS.find((user) => user.id === userId);
  return member && "avatarUrl" in member ? member.avatarUrl : undefined;
}

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
    id: "mcp-client-cursor-arnold",
    clientName: "Cursor",
    userId: ARNOLD_USER.id,
    userName: ARNOLD_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(ARNOLD_USER.id),
    createdAt: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
  },
  {
    id: "mcp-client-cursor-jamie",
    clientName: "Cursor",
    userId: OPERATOR_USER.id,
    userName: OPERATOR_USER.name,
    userAvatarUrl: mcpClientUserAvatarUrl(OPERATOR_USER.id),
    createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

function superplaneMcpServerHarness(fixture: FactoriesFixture) {
  return (
    <FactoriesHarness
      pathSuffix={superplaneMcpServerPath}
      experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      factoriesFixture={fixture}
    />
  );
}

/** Zero state with illustration (no inbound clients). */
export const Empty: Story = {
  render: () => superplaneMcpServerHarness(defaultFactoriesFixture),
};

/** Inbound MCP clients connected. */
export const Configured: Story = {
  render: () =>
    superplaneMcpServerHarness({
      ...defaultFactoriesFixture,
      mcpClientsByFactoryId: { [PRIMARY_FACTORY_ID]: CONFIGURED_MCP_CLIENTS },
    }),
};
