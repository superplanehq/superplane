import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { MockConsoleProvider, PanelFrame } from "../__stories__/storyDecorators";
import {
  columnFormatShowcasePanelSize,
  columnFormatShowcaseRender,
  columnFormatShowcaseRows,
  executionRows,
  prRiskCheckRows,
  prRiskChecksPanelSize,
  prRiskChecksTableRender,
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

/** Wide panel with one column per supported format (text, status, badge, number, percent, duration, date, datetime, relative, code, link). */
export const ColumnFormatShowcase: Story = {
  parameters: { layout: "padded" },
  render: (args) => (
    <TablePanel
      title="Column format showcase"
      width={columnFormatShowcasePanelSize.width}
      height={columnFormatShowcasePanelSize.height}
      {...args}
    />
  ),
  args: {
    render: columnFormatShowcaseRender,
    rows: columnFormatShowcaseRows,
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

/** `avatar` format: circular profile images with an icon fallback for a missing URL. */
export const Avatars: Story = {
  render: (args) => <TablePanel title="Contributors" {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "avatarUrl", label: "", format: "avatar" },
        { field: "author", label: "Author" },
        { field: "commits", label: "Commits", format: "number" },
      ],
    },
    rows: [
      { id: "c1", author: "torvalds", avatarUrl: "https://github.com/torvalds.png", commits: 1284 },
      { id: "c2", author: "gaearon", avatarUrl: "https://github.com/gaearon.png", commits: 512 },
      { id: "c3", author: "sindresorhus", avatarUrl: "https://github.com/sindresorhus.png", commits: 377 },
      { id: "c4", author: "unknown", avatarUrl: "", commits: 12 },
    ],
    isLoading: false,
  },
};

/**
 * Realistic combo: an engineering throughput board pairing `avatar` (contributor
 * photo) with `number`, `progress` (review coverage), and `trend` (week-over-week,
 * higher is better so a rise is green).
 */
export const TeamLeaderboard: Story = {
  parameters: { layout: "padded" },
  render: (args) => <TablePanel title="Team throughput (this sprint)" width={640} height={260} {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "avatarUrl", label: "", format: "avatar" },
        { field: "engineer", label: "Engineer" },
        { field: "prsMerged", label: "PRs merged", format: "number" },
        { field: "reviewCoverage", label: "Review coverage", format: "progress" },
        { field: "weekOverWeek", label: "WoW", format: "trend", goodDirection: "up" },
      ],
      sort: { field: "prsMerged", order: "desc" },
    },
    rows: [
      {
        id: "e1",
        engineer: "torvalds",
        avatarUrl: "https://github.com/torvalds.png",
        prsMerged: 42,
        reviewCoverage: 0.94,
        weekOverWeek: 0.18,
      },
      {
        id: "e2",
        engineer: "gaearon",
        avatarUrl: "https://github.com/gaearon.png",
        prsMerged: 37,
        reviewCoverage: 0.88,
        weekOverWeek: 0.05,
      },
      {
        id: "e3",
        engineer: "sindresorhus",
        avatarUrl: "https://github.com/sindresorhus.png",
        prsMerged: 29,
        reviewCoverage: 0.76,
        weekOverWeek: -0.12,
      },
      {
        id: "e4",
        engineer: "yyx990803",
        avatarUrl: "https://github.com/yyx990803.png",
        prsMerged: 24,
        reviewCoverage: 0.91,
        weekOverWeek: 0,
      },
      {
        id: "e5",
        engineer: "kentcdodds",
        avatarUrl: "https://github.com/kentcdodds.png",
        prsMerged: 18,
        reviewCoverage: 0.63,
        weekOverWeek: 0.22,
      },
    ],
    isLoading: false,
  },
};

/**
 * Realistic combo: an SRE reliability board. `badge` (service) + `avatar` (owner)
 * + `status` + `progress` (30-day uptime) + `trend` where lower is better
 * (`goodDirection: "down"`, so a falling error rate paints green) + `duration` (p95).
 */
export const ServiceReliability: Story = {
  parameters: { layout: "padded" },
  render: (args) => <TablePanel title="Service reliability" width={760} height={260} {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "service", label: "Service", format: "badge" },
        { field: "ownerAvatar", label: "Owner", format: "avatar" },
        { field: "status", label: "Status", format: "status" },
        { field: "uptime", label: "Uptime (30d)", format: "progress" },
        { field: "errorRateChange", label: "Error rate Δ", format: "trend", goodDirection: "down" },
        { field: "p95Ms", label: "p95", format: "duration" },
      ],
      rowStyles: [{ field: "status", op: "eq", value: "failed", tone: "red-soft" }],
    },
    rows: [
      {
        id: "s1",
        service: "api-gateway",
        ownerAvatar: "https://github.com/gaearon.png",
        status: "passed",
        uptime: 0.999,
        errorRateChange: -0.34,
        p95Ms: 120,
      },
      {
        id: "s2",
        service: "checkout",
        ownerAvatar: "https://github.com/torvalds.png",
        status: "running",
        uptime: 0.982,
        errorRateChange: 0.12,
        p95Ms: 340,
      },
      {
        id: "s3",
        service: "search",
        ownerAvatar: "https://github.com/sindresorhus.png",
        status: "passed",
        uptime: 0.9995,
        errorRateChange: -0.08,
        p95Ms: 88,
      },
      {
        id: "s4",
        service: "billing",
        ownerAvatar: "https://github.com/yyx990803.png",
        status: "failed",
        uptime: 0.947,
        errorRateChange: 0.41,
        p95Ms: 512,
      },
      {
        id: "s5",
        service: "notifications",
        ownerAvatar: "https://github.com/kentcdodds.png",
        status: "passed",
        uptime: 0.997,
        errorRateChange: 0,
        p95Ms: 60,
      },
    ],
    isLoading: false,
  },
};

/**
 * Realistic combo: a FinOps spend board. `avatar` (team lead) + `number`
 * (monthly spend) + `progress` (budget used) + `trend` month-over-month where a
 * drop in spend is good (`goodDirection: "down"`).
 */
export const CloudSpendByTeam: Story = {
  parameters: { layout: "padded" },
  render: (args) => <TablePanel title="Cloud spend by team" width={700} height={260} {...args} />,
  args: {
    render: {
      kind: "table",
      columns: [
        { field: "ownerAvatar", label: "", format: "avatar" },
        { field: "team", label: "Team" },
        { field: "spend", label: "Spend", format: "number" },
        { field: "budgetUsed", label: "Budget used", format: "progress" },
        { field: "monthOverMonth", label: "MoM", format: "trend", goodDirection: "down" },
      ],
      sort: { field: "spend", order: "desc" },
    },
    rows: [
      {
        id: "t1",
        team: "Platform",
        ownerAvatar: "https://github.com/torvalds.png",
        spend: 48200,
        budgetUsed: 0.96,
        monthOverMonth: 0.08,
      },
      {
        id: "t2",
        team: "Data",
        ownerAvatar: "https://github.com/sindresorhus.png",
        spend: 31500,
        budgetUsed: 0.7,
        monthOverMonth: -0.15,
      },
      {
        id: "t3",
        team: "Growth",
        ownerAvatar: "https://github.com/gaearon.png",
        spend: 18900,
        budgetUsed: 0.54,
        monthOverMonth: -0.04,
      },
      {
        id: "t4",
        team: "ML",
        ownerAvatar: "https://github.com/kentcdodds.png",
        spend: 12300,
        budgetUsed: 0.41,
        monthOverMonth: 0.23,
      },
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
