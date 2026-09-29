import type { Meta, StoryObj } from "@storybook/react-vite";

import { MCPToolsList } from "./MCPToolsList";

const tools = [
  { name: "search", readOnly: true },
  { name: "list_issues", readOnly: true },
  { name: "get_issue", readOnly: true },
  { name: "list_comments", readOnly: true },
  { name: "create_issue", readOnly: false },
  { name: "update_issue", readOnly: false },
  { name: "delete_issue", readOnly: false },
  { name: "create_comment", readOnly: false },
  { name: "assign_issue", readOnly: false },
  { name: "close_issue", readOnly: false },
  { name: "reopen_issue", readOnly: false },
  { name: "add_label", readOnly: false },
];

const meta = {
  title: "Factories/Pages/Settings/MCP tools",
  component: MCPToolsList,
  parameters: { layout: "padded" },
} satisfies Meta<typeof MCPToolsList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ToolCount: Story = {
  args: {
    tools,
    isLoading: false,
    isError: false,
    disabledTools: ["create_issue", "update_issue", "delete_issue", "add_label"],
    canUpdate: true,
    onToggleTool: () => undefined,
  },
};

export const WriteFirst: Story = {
  args: {
    tools,
    isLoading: false,
    isError: false,
    disabledTools: [],
    canUpdate: true,
    onToggleTool: () => undefined,
    initialSort: "write",
  },
};

export const ReadFirst: Story = {
  args: {
    tools,
    isLoading: false,
    isError: false,
    disabledTools: [],
    canUpdate: true,
    onToggleTool: () => undefined,
    initialSort: "read",
  },
};
