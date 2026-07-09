import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame } from "../__stories__/storyDecorators";
import { WidgetRanking } from "./WidgetRanking";

/**
 * Ranking / leaderboard panel (prototype). Groups execution-shaped rows by a
 * field, aggregates a metric per group, sorts into a ranking, and shows an
 * optional trend column comparing the current rolling window to the previous
 * one. Data flows in as plain rows (same shape as the other console widgets),
 * so the fixtures below mirror `executions` rows: `nodeName`, `status`,
 * `durationMs`, `createdAt`.
 */
const meta = {
  title: "Console/Ranking",
  component: WidgetRanking,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetRanking>;

export default meta;
type Story = StoryObj<typeof meta>;

const DAY_MS = 24 * 60 * 60 * 1000;

/** ISO timestamp `days` before now, so trend windows resolve against the wall clock. */
function daysAgo(days: number): string {
  return new Date(Date.now() - days * DAY_MS).toISOString();
}

/**
 * Generate execution-shaped rows for one node, spread across the current
 * (0–7d) and previous (7–14d) windows so the 7d trend column has a baseline
 * to compare against.
 */
function nodeExecutions(
  nodeName: string,
  opts: { current: number; previous: number; durationMs: number; status?: string },
): Record<string, unknown>[] {
  const rows: Record<string, unknown>[] = [];
  const status = opts.status ?? "passed";
  for (let i = 0; i < opts.current; i++) {
    rows.push({
      id: `${nodeName}-cur-${i}`,
      nodeName,
      status,
      durationMs: opts.durationMs,
      createdAt: daysAgo(1 + (i % 6)),
    });
  }
  for (let i = 0; i < opts.previous; i++) {
    rows.push({
      id: `${nodeName}-prev-${i}`,
      nodeName,
      status,
      durationMs: opts.durationMs,
      createdAt: daysAgo(8 + (i % 6)),
    });
  }
  return rows;
}

/** Mixed trend fixture: up, down, flat, and a brand-new entrant. */
const executionRows: Record<string, unknown>[] = [
  ...nodeExecutions("deploy-prod", { current: 9, previous: 5, durationMs: 42_000 }),
  ...nodeExecutions("run-tests", { current: 3, previous: 7, durationMs: 65_000, status: "failed" }),
  ...nodeExecutions("build-image", { current: 4, previous: 4, durationMs: 128_000 }),
  ...nodeExecutions("lint", { current: 6, previous: 0, durationMs: 3_500 }),
  ...nodeExecutions("notify-slack", { current: 2, previous: 3, durationMs: 1_200 }),
];

function RankingPanel({
  title,
  height = 300,
  ...props
}: { title?: string; height?: number } & ComponentProps<typeof WidgetRanking>) {
  return (
    <PanelFrame height={height} width={460}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Ranking"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetRanking {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

export const CountByNode: Story = {
  render: (args) => <RankingPanel title="Executions by node (7d)" {...args} />,
  args: {
    render: {
      kind: "ranking",
      groupField: "nodeName",
      groupLabel: "Node",
      aggregation: "count",
      label: "Runs",
      format: "number",
      trend: { timestampField: "createdAt", window: "7d" },
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const SumDurationByNode: Story = {
  render: (args) => <RankingPanel title="Total run time by node (7d)" {...args} />,
  args: {
    render: {
      kind: "ranking",
      groupField: "nodeName",
      groupLabel: "Node",
      aggregation: "sum",
      valueField: "durationMs",
      label: "Total time",
      format: "duration",
      trend: { timestampField: "createdAt", window: "7d" },
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const NoTrend: Story = {
  render: (args) => <RankingPanel title="Executions by node" {...args} />,
  args: {
    render: {
      kind: "ranking",
      groupField: "nodeName",
      groupLabel: "Node",
      aggregation: "count",
      label: "Runs",
      format: "number",
    },
    rows: executionRows,
    isLoading: false,
  },
};

export const Empty: Story = {
  render: (args) => <RankingPanel title="Executions by node (7d)" {...args} />,
  args: {
    render: {
      kind: "ranking",
      groupField: "nodeName",
      groupLabel: "Node",
      aggregation: "count",
      label: "Runs",
      trend: { timestampField: "createdAt", window: "7d" },
    },
    rows: [],
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <RankingPanel title="Executions by node (7d)" {...args} />,
  args: {
    render: {
      kind: "ranking",
      groupField: "nodeName",
      groupLabel: "Node",
      aggregation: "count",
      label: "Runs",
      trend: { timestampField: "createdAt", window: "7d" },
    },
    rows: [],
    isLoading: true,
  },
};
