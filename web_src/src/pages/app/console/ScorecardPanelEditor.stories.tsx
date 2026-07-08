import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@/components/ui/button";

import { ScorecardPanelEditor } from "./ScorecardPanelEditor";
import { DEFAULT_SCORECARD_CONTENT, type ScorecardPanelContent } from "./scorecardContent";
import {
  cloudSpendByAccount,
  fetchDurationsMs,
  fetchTimeBands,
  papercutOpenBands,
  papercutTotalOpen,
} from "./__stories__/storyFixtures";

/**
 * Ground-up edit experience for the scorecard panel — a self-contained replica
 * of the real panel editor modal (header, Form/YAML tabs, Save/Cancel) with an
 * always-on live preview.
 *
 * The stories are seeded with real data from live canvases in the org, chosen
 * to exercise every data source:
 * - Runs — the "Papercut Analysis" canvas, reading its `Format Message` node
 *   output via the per-node `$` map (e.g. `$["Format Message"].data.result.total_open`).
 * - Memory — the "LLM Cost Tracker 2" `aws_costs_by_account` namespace.
 * - Executions — the "Papercut Analysis" `Fetch GitHub Stats` node durations.
 */
const meta = {
  title: "Console/Scorecard Editor (prototype)",
  component: ScorecardPanelEditor,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof ScorecardPanelEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

/** Path into a run row that reaches the Format Message node's latest output. */
const FMT = '$["Format Message"].data.result';

/** Shared base: sensible resets on top of the editor default. */
const base: ScorecardPanelContent = {
  ...DEFAULT_SCORECARD_CONTENT,
  prefix: "",
  suffix: "",
  trendLabel: "vs start of range",
};

function EditorHarness({
  initialContent,
  sampleSeries,
}: {
  initialContent: ScorecardPanelContent;
  sampleSeries: number[];
}) {
  const [open, setOpen] = useState(true);
  const [content, setContent] = useState<ScorecardPanelContent>(initialContent);
  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-8 dark:bg-gray-950">
      <Button type="button" onClick={() => setOpen(true)}>
        Open scorecard editor
      </Button>
      <ScorecardPanelEditor
        open={open}
        onOpenChange={setOpen}
        initialContent={content}
        onSave={(next) => setContent(next)}
        sampleSeries={sampleSeries}
      />
    </div>
  );
}

/** Runs — open UX papercuts, the hero metric. Lower is better; scored against a target. */
export const OpenPapercuts: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={papercutTotalOpen}
      initialContent={{
        ...base,
        dataSource: { kind: "runs", limit: 30 },
        aggregation: "last",
        field: `${FMT}.total_open`,
        format: "number",
        label: "Open UX papercuts",
        goalDirection: "lower",
        statusMode: "target",
        target: 100,
        seriesField: `${FMT}.total_open`,
        showSparkline: true,
        showTrend: true,
      }}
    />
  ),
};

/** Runs — same metric, scored against threshold bands instead of a single target. */
export const OpenPapercutsBands: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={papercutTotalOpen}
      initialContent={{
        ...base,
        dataSource: { kind: "runs", limit: 30 },
        aggregation: "last",
        field: `${FMT}.total_open`,
        format: "number",
        label: "Open UX papercuts",
        goalDirection: "lower",
        statusMode: "thresholds",
        thresholds: papercutOpenBands,
        target: undefined,
        seriesField: `${FMT}.total_open`,
        showSparkline: true,
        showTrend: true,
      }}
    />
  ),
};

/** Memory — total daily cloud spend, summed across account records, with progress to a budget. */
export const DailyCloudSpend: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={cloudSpendByAccount}
      initialContent={{
        ...base,
        dataSource: { kind: "memory", namespace: "aws_costs_by_account" },
        aggregation: "sum",
        field: "cost_usd",
        format: "number",
        prefix: "$",
        label: "Daily cloud spend",
        goalDirection: "lower",
        statusMode: "target",
        target: 100,
        showProgress: true,
        seriesField: "cost_usd",
        showSparkline: true,
        showTrend: false,
      }}
    />
  ),
};

/** Executions — average GitHub fetch time for one node, formatted as a duration. */
export const AvgFetchTime: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={fetchDurationsMs}
      initialContent={{
        ...base,
        dataSource: { kind: "executions", node: "fetch-github-stats", limit: 100 },
        aggregation: "avg",
        field: "durationMs",
        format: "duration",
        label: "Avg GitHub fetch time",
        goalDirection: "lower",
        statusMode: "thresholds",
        thresholds: fetchTimeBands,
        target: undefined,
        seriesField: "durationMs",
        showSparkline: true,
        showTrend: true,
      }}
    />
  ),
};

/** Executions — a plain count of report runs; no field or series needed. */
export const DailyReports: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={papercutTotalOpen}
      initialContent={{
        ...base,
        dataSource: { kind: "executions", node: "format-message", limit: 200 },
        aggregation: "count",
        field: undefined,
        format: "number",
        label: "Daily reports",
        goalDirection: "higher",
        statusMode: "target",
        target: 7,
        seriesField: undefined,
        showSparkline: false,
        showTrend: false,
      }}
    />
  ),
};

/** Invalid: target mode with no target set — shows the inline validation. */
export const Invalid: Story = {
  render: () => (
    <EditorHarness
      sampleSeries={papercutTotalOpen}
      initialContent={{
        ...base,
        dataSource: { kind: "runs", limit: 30 },
        aggregation: "last",
        field: `${FMT}.total_open`,
        format: "number",
        label: "Open UX papercuts",
        goalDirection: "lower",
        statusMode: "target",
        target: undefined,
        seriesField: `${FMT}.total_open`,
      }}
    />
  ),
};
