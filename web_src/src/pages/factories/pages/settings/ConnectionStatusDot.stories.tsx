import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
} from "../../__fixtures__/agentResourceFixtures";
import { ConnectionStatusDot } from "./ConnectionStatusDot";

const meta = {
  title: "Factories/Pages/Settings/MCP status",
  component: ConnectionStatusDot,
  parameters: { layout: "padded" },
} satisfies Meta<typeof ConnectionStatusDot>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ConnectedGreenDot: Story = {
  args: { resource: OAUTH_CONNECTED_RESOURCE },
};

export const HeaderReady: Story = {
  args: { resource: HEADER_MCP_RESOURCE },
};

export const NotConnected: Story = {
  args: { resource: OAUTH_NOT_CONNECTED_RESOURCE },
};
