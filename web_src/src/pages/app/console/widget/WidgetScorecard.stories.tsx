import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame } from "../__stories__/storyDecorators";
import { WidgetScorecard } from "./WidgetScorecard";

/**
 * Scorecard panel renderer. Displays a single KPI value with an optional
 * status-colored change chip (vs the immediately previous value in the
 * series), target-driven progress bar, and status-colored sparkline.
 *
 * Every story below feeds the real `WidgetScorecard` component through the
 * same `TypedPanelShell` chrome the live console uses — no fake / mock
 * scorecard exists. Rows are static per-story fixtures so the widget
 * exercises the same `aggregateNumber` + `extractScorecardSeries` +
 * `pickChangeAnchors` + `resolveScorecardTarget` pipeline it runs against
 * real data.
 */
const meta = {
  title: "Console/Scorecard",
  component: WidgetScorecard,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetScorecard>;

export default meta;
type Story = StoryObj<typeof meta>;

// Rows mimic the screenshot from the PRD: an "openCount" that starts at
// 127 and drops to 98. With the new "vs previous" semantics the change
// chip compares the last row (98) to the immediately previous one (105)
// → a green -7 for a "lower is better" direction.
const openCountRows = [
  { openCount: 127 },
  { openCount: 120 },
  { openCount: 118 },
  { openCount: 112 },
  { openCount: 110 },
  { openCount: 105 },
  { openCount: 98 },
];

// Same-shape series that trends the *wrong* way, so a `better: down` KPI
// (errors) or a `better: up` KPI configured on this data lights up red.
const errorGrowingRows = [
  { errors: 6 },
  { errors: 8 },
  { errors: 11 },
  { errors: 14 },
  { errors: 18 },
  { errors: 23 },
  { errors: 29 },
];

// Flat series — every point equals the baseline. Change chip renders in
// the muted "flat" gray with a zero delta.
const flatRows = Array.from({ length: 6 }, () => ({ signups: 42 }));

const revenueRows = [
  { mrr: 12_000 },
  { mrr: 13_500 },
  { mrr: 15_800 },
  { mrr: 15_100 },
  { mrr: 16_400 },
  { mrr: 17_900 },
  { mrr: 19_200 },
];

// SLA rows so we can hit a `better: up` target both when we hit it (last
// row met the 99.5% SLA) and use the same data for overshoot / miss
// variants by tweaking the target string alone.
const slaRows = [
  { passRate: 0.982 },
  { passRate: 0.985 },
  { passRate: 0.988 },
  { passRate: 0.991 },
  { passRate: 0.993 },
  { passRate: 0.994 },
  { passRate: 0.996 },
];

// Rows that carry the target inline so `resolveScorecardTarget` can pull
// it from the last row via a CEL expression (`{{ target }}`). This is the
// realistic shape for memory-backed KPIs where the goal ships alongside
// the metric.
const revenueWithTargetRows = [
  { mrr: 12_000, target: 18_000 },
  { mrr: 13_500, target: 18_000 },
  { mrr: 15_800, target: 18_000 },
  { mrr: 15_100, target: 18_000 },
  { mrr: 16_400, target: 18_000 },
  { mrr: 17_900, target: 18_000 },
  { mrr: 19_200, target: 18_000 },
];

function ScorecardPanel({
  title,
  height = 220,
  ...props
}: { title?: string; height?: number } & ComponentProps<typeof WidgetScorecard>) {
  return (
    <PanelFrame height={height}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Scorecard"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetScorecard {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

/**
 * PRD screenshot state: "Open UX papercuts" trending down against a target
 * of 80. `better: down` + shrinking value → emerald change chip, sparkline,
 * and progress bar all in the "better" color.
 */
export const OpenPapercutsShrinking: Story = {
  render: (args) => <ScorecardPanel title="Open UX papercuts" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Open UX papercuts",
      format: "number",
      better: "down",
      target: "80",
      showProgress: true,
      sparklineField: "openCount",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

/**
 * Higher-is-better KPI (MRR) that has grown, with a prefix currency
 * symbol. Target hit, so status is emerald and the progress bar caps at
 * 100% with a `percent > 100` label.
 */
export const HigherIsBetter: Story = {
  render: (args) => <ScorecardPanel title="Monthly recurring revenue" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "mrr",
      label: "MRR",
      format: "number",
      prefix: "R$",
      better: "up",
      target: "18000",
      showProgress: true,
      sparklineField: "mrr",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: revenueRows,
    isLoading: false,
  },
};

/**
 * Errors going *up* against a `better: down` goal. The change chip should
 * render in red with an up arrow, the sparkline turns red, and if a target
 * is configured (below) the progress bar also flips to worse.
 */
export const WorseThanBaseline: Story = {
  render: (args) => <ScorecardPanel title="Errors today" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "errors",
      label: "Errors",
      format: "number",
      better: "down",
      target: "10",
      showProgress: true,
      sparklineField: "errors",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: errorGrowingRows,
    isLoading: false,
  },
};

/**
 * Baseline equals current value — the change chip renders the muted
 * "flat" state (gray `- 0`) and the sparkline uses the neutral tone
 * instead of emerald/red. Target still resolves so the progress bar
 * remains visible in its target-based color.
 */
export const FlatChange: Story = {
  render: (args) => <ScorecardPanel title="Signups this week" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "signups",
      label: "Signups",
      format: "number",
      better: "up",
      target: "40",
      showProgress: true,
      sparklineField: "signups",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: flatRows,
    isLoading: false,
  },
};

/**
 * `better: up` target overshoot — the current value beats target by
 * more than 100%. Progress bar clamps at 100%, but the numeric label
 * still shows the raw percent (e.g. `240% of 80`) so overshoot is
 * visible without breaking layout.
 */
export const TargetOvershoot: Story = {
  render: (args) => <ScorecardPanel title="Signups this week" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "sum",
      field: "signups",
      label: "Signups",
      format: "number",
      better: "up",
      target: "80",
      showProgress: true,
      showChange: "none",
      changeCaption: "vs target",
    },
    rows: flatRows,
    isLoading: false,
  },
};

/**
 * Single-point series → no change baseline. Widget falls back to the
 * target for status coloring. Here the SLA hits the 99.5% target, so
 * the panel is emerald with a full progress bar.
 */
export const TargetHitNoSeries: Story = {
  render: (args) => <ScorecardPanel title="Pass rate (last hour)" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "avg",
      field: "passRate",
      label: "Pass rate",
      format: "percent",
      better: "up",
      target: "0.995",
      showProgress: true,
    },
    rows: [slaRows[slaRows.length - 1]],
    isLoading: false,
  },
};

/**
 * Same shape as `TargetHitNoSeries` but the target isn't met. Change is
 * still absent (single-point series), so the widget colors status via the
 * target comparison — red bar and status dot.
 */
export const TargetMissedNoSeries: Story = {
  render: (args) => <ScorecardPanel title="Pass rate (last hour)" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "avg",
      field: "passRate",
      label: "Pass rate",
      format: "percent",
      better: "up",
      target: "0.999",
      showProgress: true,
    },
    rows: [{ passRate: 0.982 }],
    isLoading: false,
  },
};

/**
 * `count` aggregation over rows — no `field` needed. Useful when the
 * scorecard tracks how many events matched the filters (e.g. failed
 * runs today) rather than aggregating a numeric column.
 */
export const CountAggregation: Story = {
  render: (args) => <ScorecardPanel title="Failed runs today" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "count",
      label: "Failed runs",
      better: "down",
      target: "5",
      showProgress: true,
    },
    rows: Array.from({ length: 3 }, () => ({})),
    isLoading: false,
  },
};

/**
 * `showChange: "percent"` — only the percent magnitude is rendered
 * alongside the arrow (`-22.8%`). Useful when the absolute number is
 * meaningless without units.
 */
export const PercentOnlyChange: Story = {
  render: (args) => <ScorecardPanel title="Deploy failures" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Failed deploys",
      format: "number",
      better: "down",
      sparklineField: "openCount",
      showChange: "percent",
      changeCaption: "vs previous",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

/**
 * `showChange: "number"` — only the absolute delta is rendered
 * (`-29`). Useful when percent framing is confusing (e.g. counts of
 * incidents, tickets closed).
 */
export const NumberOnlyChange: Story = {
  render: (args) => <ScorecardPanel title="Incidents open" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Open incidents",
      format: "number",
      better: "down",
      sparklineField: "openCount",
      showChange: "number",
      changeCaption: "vs previous",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

/**
 * `showChange: "none"` — the change chip renders arrow-only (no
 * magnitude text). Colors and tooltip still communicate direction /
 * percent; caption is preserved.
 */
export const ArrowOnlyChange: Story = {
  render: (args) => <ScorecardPanel title="Deploy failures" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Failed deploys",
      format: "number",
      better: "down",
      sparklineField: "openCount",
      showChange: "none",
      changeCaption: "vs previous",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

/**
 * Value with a *trailing* suffix (like " MWh"). Prefix stays flush,
 * suffix is spaced from the value and rendered a tick smaller so the
 * primary number keeps visual dominance. Uses a shrinking energy curve
 * against `better: down` so the surrounding chrome renders in the
 * emerald "better" state, isolating suffix rendering visually.
 */
export const WithSuffix: Story = {
  render: (args) => <ScorecardPanel title="Energy usage" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "kwh",
      label: "Energy this month",
      format: "number",
      suffix: " MWh",
      better: "down",
      target: "20",
      showProgress: true,
      sparklineField: "kwh",
      showChange: "both",
      changeCaption: "vs previous",
    },
    // Reverse the revenue curve so consumption trends *down*.
    rows: [...revenueRows].reverse().map((r) => ({ kwh: r.mrr / 1000 })),
    isLoading: false,
  },
};

/**
 * Prefix + suffix combo (currency prefix `$`, per-month suffix). Both
 * flank the aggregated value on the same baseline so the block reads as
 * a single unit.
 */
export const PrefixAndSuffix: Story = {
  render: (args) => <ScorecardPanel title="Spend this month" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "mrr",
      label: "Spend",
      format: "number",
      prefix: "$",
      suffix: " /mo",
      better: "down",
      sparklineField: "mrr",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: revenueRows,
    isLoading: false,
  },
};

/**
 * Sparkline explicitly disabled (no `sparklineField`), and the aggregation
 * (`count`) has no primary `field` to fall back to — so the change chip
 * hides too. Status coloring falls back to the target.
 */
export const NoSparkline: Story = {
  render: (args) => <ScorecardPanel title="Deploys this week" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "count",
      label: "Deploys",
      better: "up",
      target: "10",
      showProgress: true,
    },
    rows: Array.from({ length: 14 }, () => ({})),
    isLoading: false,
  },
};

/**
 * No `sparklineField`, but the primary `field` (openCount) still drives
 * the change chip. This mirrors the common runs / executions pattern
 * where authors just want the "vs previous" number without dedicating
 * extra vertical space to a sparkline.
 */
export const NoSparklineFieldStillShowsChange: Story = {
  render: (args) => <ScorecardPanel title="Open UX papercuts" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Open UX papercuts",
      format: "number",
      better: "down",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

/**
 * Target expressed as a CEL expression evaluated against the last
 * filtered row. Realistic pattern for memory-backed KPIs where the goal
 * ships in the same document as the metric.
 */
export const CelTarget: Story = {
  render: (args) => <ScorecardPanel title="MRR vs goal" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "mrr",
      label: "MRR vs goal",
      format: "number",
      prefix: "$",
      better: "up",
      target: "{{ target }}",
      showProgress: true,
      sparklineField: "mrr",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: revenueWithTargetRows,
    isLoading: false,
  },
};

/**
 * Target expressed as a bare dot path resolved against the last row
 * (no `{{ }}` syntax). Same result as the CEL variant — just the
 * simpler author affordance for row-relative targets.
 */
export const DotPathTarget: Story = {
  render: (args) => <ScorecardPanel title="MRR vs goal" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "mrr",
      label: "MRR vs goal",
      format: "number",
      prefix: "$",
      better: "up",
      target: "target",
      showProgress: true,
      sparklineField: "mrr",
      showChange: "both",
      changeCaption: "vs previous",
    },
    rows: revenueWithTargetRows,
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <ScorecardPanel title="Open UX papercuts" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Open UX papercuts",
    },
    rows: [],
    isLoading: true,
  },
};

export const Empty: Story = {
  render: (args) => <ScorecardPanel title="Open UX papercuts" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "last",
      field: "openCount",
      label: "Open UX papercuts",
    },
    rows: [],
    isLoading: false,
  },
};
