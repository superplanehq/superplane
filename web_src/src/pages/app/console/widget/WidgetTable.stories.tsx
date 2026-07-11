import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { MockConsoleProvider, PanelFrame } from "../__stories__/storyDecorators";
import {
  executionRows,
  prRiskCheckRows,
  prRiskChecksPanelSize,
  prRiskChecksTableRender,
  progressRows,
} from "../__stories__/storyFixtures";
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
  decorators: [
    (Story) => (
      <MockConsoleProvider>
        <Story />
      </MockConsoleProvider>
    ),
  ],
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

const avatarRows: Record<string, unknown>[] = [
  {
    id: "u-1",
    name: "Ada Lovelace",
    role: "Owner",
    avatarUrl: "https://i.pravatar.cc/64?img=47",
  },
  {
    id: "u-2",
    name: "Grace Hopper",
    role: "Maintainer",
    avatarUrl: "https://i.pravatar.cc/64?img=32",
  },
  {
    id: "u-3",
    name: "Alan Turing",
    role: "Contributor",
    avatarUrl: "https://i.pravatar.cc/64?img=12",
  },
  {
    id: "u-4",
    name: "Katherine Johnson",
    role: "Contributor",
    avatarUrl: "",
  },
  {
    // A bare GitHub username resolves to the github.com avatar with the
    // username in the tooltip (see resolveConsoleAvatar).
    id: "u-5",
    name: "GitHub username",
    role: "Guest",
    avatarUrl: "torvalds",
  },
];

export const Avatars: Story = {
  render: (args) => <TablePanel title="Team roster" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "avatarUrl", label: "", format: "avatar" },
        { field: "name", label: "Name" },
        { field: "role", label: "Role", format: "badge" },
      ],
    },
    rows: avatarRows,
    isLoading: false,
  },
};

export const ManyColumnsAndFormats: Story = {
  render: (args) => <TablePanel title="All column formats" width={880} {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "name", label: "Node" },
        { field: "status", label: "Status", format: "status" },
        { field: "service", label: "Service", format: "badge" },
        { field: "durationMs", label: "Duration", format: "duration" },
        { field: "cost", label: "Cost", format: "number" },
        {
          field: "cost",
          label: "Budget",
          format: "progress",
          progressTarget: "cost_budget",
          progressLabel: "number",
        },
        { field: "createdAt", label: "Started", format: "datetime" },
        { field: "url", label: "Link", format: "link" },
        { field: '{{ "https://i.pravatar.cc/64?u=" + id }}', label: "Avatar", format: "avatar" },
      ],
    },
    rows: executionRows,
    isLoading: false,
  },
};

/**
 * Every timestamp-formatted column renders through the shared `Timestamp`
 * component — each label carries a dashed underline hint, and hovering any
 * cell reveals Local / UTC / Relative / ISO with a copy affordance for the
 * ISO value.
 */
export const TimestampFormats: Story = {
  render: (args) => <TablePanel title="Timestamp column formats" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "name", label: "Node" },
        { field: "createdAt", label: "Day", format: "date" },
        { field: "createdAt", label: "When", format: "datetime" },
        { field: "createdAt", label: "Started", format: "relative" },
      ],
    },
    rows: executionRows,
    isLoading: false,
  },
};

/**
 * Dedicated showcase for the `progress` column format. Rows cover the mid,
 * near-full, overshoot, zero, and empty branches so the bar clamping, label
 * modes, and em-dash placeholder are all visible in a single view.
 */
export const ProgressColumn: Story = {
  render: (args) => <TablePanel title="Progress column showcase" width={720} {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "label", label: "Row" },
        {
          field: "done",
          label: "Percent",
          format: "progress",
          progressTarget: "total",
          progressLabel: "percent",
        },
        {
          field: "done",
          label: "Number",
          format: "progress",
          progressTarget: "total",
          progressLabel: "number",
        },
        {
          field: "done",
          label: "Bar only",
          format: "progress",
          progressTarget: "total",
          progressLabel: "none",
        },
      ],
    },
    rows: progressRows,
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

/**
 * Rows are pre-ordered newest-first so each cell compares against the row
 * below (its predecessor in time). Covers every trend edge state:
 * - Increases and decreases in both polarity modes
 * - A flat delta (`- 0`)
 * - The last row with no baseline (`- 0`)
 */
const trendRows: Record<string, unknown>[] = [
  { deploy: "deploy #106", durationMs: 4200, errorRate: 0.4, throughput: 1250 },
  { deploy: "deploy #105", durationMs: 5100, errorRate: 0.8, throughput: 1180 },
  { deploy: "deploy #104", durationMs: 5100, errorRate: 0.8, throughput: 1180 },
  { deploy: "deploy #103", durationMs: 4700, errorRate: 1.2, throughput: 940 },
  { deploy: "deploy #102", durationMs: 6800, errorRate: 2.5, throughput: 720 },
];

export const Trend: Story = {
  render: (args) => <TablePanel title="Deploys" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "deploy", label: "Deploy" },
        { field: "durationMs", label: "Duration", format: "duration" },
        {
          field: "durationMs",
          label: "Duration Δ",
          format: "trend",
          trendBetter: "down",
          trendDisplay: "percent",
        },
        { field: "errorRate", label: "Errors %", format: "number" },
        {
          field: "errorRate",
          label: "Errors Δ",
          format: "trend",
          trendBetter: "down",
          trendDisplay: "value",
        },
        { field: "throughput", label: "RPS", format: "number" },
        {
          field: "throughput",
          label: "RPS Δ",
          format: "trend",
          trendBetter: "up",
          trendDisplay: "percent",
        },
      ],
    },
    rows: trendRows,
    isLoading: false,
  },
};

/**
 * Same rows as the Trend story, but the table advertises `hasMore: true`
 * with no peek row — the last cell renders `...` while it waits for a
 * predecessor that has not been fetched yet.
 */
export const TrendPending: Story = {
  render: (args) => <TablePanel title="Deploys (loading more)" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "deploy", label: "Deploy" },
        { field: "durationMs", label: "Duration", format: "duration" },
        {
          field: "durationMs",
          label: "Duration Δ",
          format: "trend",
          trendBetter: "down",
          trendDisplay: "percent",
        },
      ],
    },
    rows: trendRows,
    isLoading: false,
    hasMore: true,
    isFetchingMore: false,
    onLoadMore: () => console.log("load more"),
  },
};

/**
 * `hasMore` is true only because more rows are already loaded behind the
 * progressive display window. Pass the full loaded set + `displayCount` so
 * the last visible trend cell compares against the hidden neighbor.
 */
export const TrendDisplayWindowPeek: Story = {
  render: (args) => <TablePanel title="Deploys (more loaded)" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "deploy", label: "Deploy" },
        { field: "durationMs", label: "Duration", format: "duration" },
        {
          field: "durationMs",
          label: "Duration Δ",
          format: "trend",
          trendBetter: "down",
          trendDisplay: "percent",
        },
      ],
    },
    rows: trendRows,
    displayCount: 4,
    isLoading: false,
    hasMore: true,
    isFetchingMore: false,
    onLoadMore: () => console.log("load more"),
  },
};

/**
 * Value + trend in one column via `showTrend` on number / percent / duration.
 * Same deploy rows as the Trend story — compare side-by-side with Trend.
 */
export const ValueWithTrend: Story = {
  render: (args) => <TablePanel title="Deploys (value + trend)" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "deploy", label: "Deploy" },
        {
          field: "durationMs",
          label: "Duration",
          format: "duration",
          showTrend: true,
          trendBetter: "down",
          trendDisplay: "percent",
        },
        {
          field: "errorRate",
          label: "Errors %",
          format: "number",
          showTrend: true,
          trendBetter: "down",
          trendDisplay: "value",
        },
        {
          field: "throughput",
          label: "RPS",
          format: "number",
          showTrend: true,
          trendBetter: "up",
          trendDisplay: "percent",
        },
        {
          field: "passRate",
          label: "Pass rate",
          format: "percent",
          showTrend: true,
          trendBetter: "up",
          trendDisplay: "percent",
        },
      ],
    },
    rows: [
      { deploy: "deploy #106", durationMs: 4200, errorRate: 0.4, throughput: 1250, passRate: 0.96 },
      { deploy: "deploy #105", durationMs: 5100, errorRate: 0.8, throughput: 1180, passRate: 0.93 },
      { deploy: "deploy #104", durationMs: 5100, errorRate: 0.8, throughput: 1180, passRate: 0.93 },
      { deploy: "deploy #103", durationMs: 4700, errorRate: 1.2, throughput: 940, passRate: 0.88 },
      { deploy: "deploy #102", durationMs: 6800, errorRate: 2.5, throughput: 720, passRate: 0.81 },
    ],
    isLoading: false,
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
