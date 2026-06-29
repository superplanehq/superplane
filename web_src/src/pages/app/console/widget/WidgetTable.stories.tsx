import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import {
  PanelFrame,
  executionRows,
  prRiskCheckRows,
  prRiskChecksPanelSize,
  prRiskChecksTableRender,
  withConsoleContext,
} from "../__stories__/storyHelpers";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

/**
 * Table panel renderer. Stories feed static `rows` + a `render` config directly,
 * wrapped in the real `TypedPanelShell` so the panel chrome (title, edit/delete)
 * matches the live dashboard. Row actions resolve through the mock console
 * context provided at the meta level.
 */
const meta = {
  title: "Console/Table",
  component: WidgetTable,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  decorators: [withConsoleContext],
  argTypes: {
    isLoading: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetTable>;

export default meta;
type Story = StoryObj<typeof meta>;

const defaultRender: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "name", label: "Node" },
    { field: "status", label: "Status", format: "status" },
    { field: "service", label: "Service", format: "badge" },
    { field: "durationMs", label: "Duration", format: "duration" },
    { field: "createdAt", label: "Started", format: "relative" },
  ],
  emptyMessage: "No executions yet.",
};

function TablePanel({
  title,
  width,
  height,
  ...props
}: { title?: string; width?: number; height?: number } & ComponentProps<typeof WidgetTable>) {
  return (
    <PanelFrame width={width} height={height}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Recent executions"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetTable {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

export const Populated: Story = {
  render: (args) => <TablePanel title="Recent executions" {...args} />,
  args: {
    render: defaultRender,
    rows: executionRows,
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <TablePanel title="Recent executions" {...args} />,
  args: {
    render: defaultRender,
    rows: [],
    isLoading: true,
  },
};

export const Empty: Story = {
  render: (args) => <TablePanel title="Recent executions" {...args} />,
  args: {
    render: defaultRender,
    rows: [],
    isLoading: false,
  },
};

export const RowStyles: Story = {
  render: (args) => <TablePanel title="Status-tinted rows" {...args} />,
  args: {
    render: {
      ...defaultRender,
      rowStyles: [
        { field: "status", op: "eq", value: "failed", tone: "red-soft" },
        { field: "status", op: "eq", value: "running", tone: "blue-soft" },
        { field: "status", op: "eq", value: "cancelled", tone: "dimmed" },
      ],
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const RowActions: Story = {
  render: (args) => <TablePanel title="With run action" {...args} />,
  args: {
    render: {
      ...defaultRender,
      rowActions: [{ kind: "trigger", node: "deploy-prod", label: "Run", icon: "play", variant: "primary" }],
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const ManyColumnsAndFormats: Story = {
  render: (args) => <TablePanel title="All column formats" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "name", label: "Node" },
        { field: "status", label: "Status", format: "status" },
        { field: "service", label: "Service", format: "badge" },
        { field: "durationMs", label: "Duration", format: "duration" },
        { field: "cost", label: "Cost", format: "number" },
        { field: "createdAt", label: "Started", format: "datetime" },
        { field: "url", label: "Link", format: "link" },
      ],
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const Pagination: Story = {
  render: (args) => <TablePanel title="Progressive loading" {...args} />,
  args: {
    render: defaultRender,
    rows: executionRows,
    isLoading: false,
    hasMore: true,
    isFetchingMore: false,
    onLoadMore: () => console.log("load more"),
  },
};

/** Org fixture: `pr-risk-review` console → `checks-table` memory panel. */
export const PrRiskRecentChecks: Story = {
  render: (args) => (
    <TablePanel
      title="Recent checks"
      width={prRiskChecksPanelSize.width}
      height={prRiskChecksPanelSize.height}
      {...args}
    />
  ),
  args: {
    render: prRiskChecksTableRender,
    rows: prRiskCheckRows,
    isLoading: false,
  },
};
