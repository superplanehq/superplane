import type { Meta, StoryObj } from "@storybook/react-vite";
import { BookOpen, Plug } from "lucide-react";

import { Button } from "./button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "./empty";

const meta = {
  title: "ui/Empty",
  component: Empty,
  parameters: { layout: "padded" },
} satisfies Meta<typeof Empty>;

export default meta;

type Story = StoryObj<typeof meta>;

export const MCPServers: Story = {
  render: () => (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Plug />
        </EmptyMedia>
        <EmptyTitle>No MCP servers yet</EmptyTitle>
        <EmptyDescription>Add an MCP server so agents can use it on every run.</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button type="button">Add MCP server</Button>
      </EmptyContent>
    </Empty>
  ),
};

export const Skills: Story = {
  render: () => (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <BookOpen />
        </EmptyMedia>
        <EmptyTitle>No skills yet</EmptyTitle>
        <EmptyDescription>Add a SKILL.md so agents can use it on every run.</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button type="button">Add skill</Button>
      </EmptyContent>
    </Empty>
  ),
};
