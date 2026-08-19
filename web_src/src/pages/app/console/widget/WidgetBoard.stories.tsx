import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { MockConsoleProvider, PanelFrame } from "../__stories__/storyDecorators";
import { WidgetBoard } from "./WidgetBoard";
import type { WidgetBoardRender } from "./types";

const meta = {
  title: "Console/Board",
  component: WidgetBoard,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <MockConsoleProvider value={{ canvasId: "" }}>
        <Story />
      </MockConsoleProvider>
    ),
  ],
  argTypes: {
    isLoading: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetBoard>;

export default meta;
type Story = StoryObj<typeof meta>;

const boardRender: WidgetBoardRender = {
  kind: "board",
  groupBy: "status",
  lanes: [
    { value: "Backlog", color: "gray" },
    { value: "In Progress", label: "Building", color: "blue" },
    { value: "Review", color: "yellow" },
    { value: "Done", color: "green" },
  ],
  card: {
    titleField: "title",
    fields: [
      { field: "owner", label: "Owner", format: "badge" },
      { field: "updatedAt", label: "Updated", format: "relative" },
    ],
  },
  sort: { field: "updatedAt", order: "desc" },
  rowActions: [{ kind: "trigger", node: "deploy-prod", label: "Advance", icon: "play" }],
  emptyMessage: "No tasks yet.",
};

const boardRows: Record<string, unknown>[] = [
  {
    id: "task-101",
    title: "Add board panel stories",
    status: "Done",
    owner: "Ada",
    updatedAt: "2026-07-21T18:40:00Z",
  },
  {
    id: "task-102",
    title: "Polish inline prompt form",
    status: "In Progress",
    owner: "Grace",
    updatedAt: "2026-07-21T19:15:00Z",
  },
  {
    id: "task-103",
    title: "Review YAML validation",
    status: "Review",
    owner: "Linus",
    updatedAt: "2026-07-21T19:05:00Z",
  },
  {
    id: "task-104",
    title: "Seed local test data",
    status: "Backlog",
    owner: "Margaret",
    updatedAt: "2026-07-21T17:30:00Z",
  },
  {
    id: "task-105",
    title: "Unmapped workflow state",
    status: "Blocked",
    owner: "Alan",
    updatedAt: "2026-07-21T16:00:00Z",
  },
];

function BoardPanel({
  title = "Delivery pipeline",
  ...props
}: { title?: string } & ComponentProps<typeof WidgetBoard>) {
  return (
    <PanelFrame width={920} height={430}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Board"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetBoard {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

export const Populated: Story = {
  render: (args) => <BoardPanel {...args} />,
  args: {
    render: boardRender,
    rows: boardRows,
    isLoading: false,
  },
};

export const WithOtherLane: Story = {
  render: (args) => <BoardPanel title="All workflow states" {...args} />,
  args: {
    render: { ...boardRender, otherLane: true },
    rows: boardRows,
    isLoading: false,
  },
};

export const Empty: Story = {
  render: (args) => <BoardPanel {...args} />,
  args: {
    render: boardRender,
    rows: [],
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <BoardPanel {...args} />,
  args: {
    render: boardRender,
    rows: [],
    isLoading: true,
  },
};
