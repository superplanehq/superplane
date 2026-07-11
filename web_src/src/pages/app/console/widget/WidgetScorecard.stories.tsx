import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame } from "../__stories__/storyDecorators";
import { WidgetScorecard } from "./WidgetScorecard";

/**
 * Scorecard panel renderer. Displays a single KPI value with an optional
 * status-colored change chip (vs the first data point in the series),
 * target-driven progress bar, and status-colored sparkline.
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
// 127 and drops to 98, so the change chip renders as a green -29 for a
// "lower is better" direction.
const openCountRows = [
  { openCount: 127 },
  { openCount: 120 },
  { openCount: 118 },
  { openCount: 112 },
  { openCount: 110 },
  { openCount: 105 },
  { openCount: 98 },
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
      changeCaption: "vs start of range",
    },
    rows: openCountRows,
    isLoading: false,
  },
};

const revenueRows = [
  { mrr: 12_000 },
  { mrr: 13_500 },
  { mrr: 15_800 },
  { mrr: 15_100 },
  { mrr: 16_400 },
  { mrr: 17_900 },
  { mrr: 19_200 },
];

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
      changeCaption: "vs start of range",
    },
    rows: revenueRows,
    isLoading: false,
  },
};

export const NoChangeSignal: Story = {
  render: (args) => <ScorecardPanel title="Errors today" {...args} />,
  args: {
    render: {
      kind: "scorecard",
      aggregation: "count",
      label: "Failed runs",
      better: "down",
      target: "5",
      showProgress: true,
    },
    rows: Array.from({ length: 3 }),
    isLoading: false,
  },
};

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
      changeCaption: "vs start of range",
    },
    rows: openCountRows,
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
