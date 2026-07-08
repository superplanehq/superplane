import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame } from "../__stories__/storyDecorators";
import {
  scorecardHigherBands,
  scorecardLowerBands,
  scorecardSparklineDown,
  scorecardSparklineUp,
} from "../__stories__/storyFixtures";

import { WidgetScorecard } from "./WidgetScorecard";

/**
 * Prototype `scorecard` panel renderer — a single KPI with threshold-driven
 * status color, a direction-aware trend delta, an optional sparkline, and an
 * optional progress-to-target bar. This is a mockup: it takes a pre-aggregated
 * `value` and is not yet wired into `panelTypes.ts`, the panel router, or the
 * backend.
 */
const meta = {
  title: "Console/Scorecard (prototype)",
  component: WidgetScorecard,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
    goalDirection: { control: "inline-radio", options: ["higher", "lower"] },
    showProgress: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetScorecard>;

export default meta;
type Story = StoryObj<typeof meta>;

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

export const OnTrack: Story = {
  render: (args) => <ScorecardPanel title="Deploy success rate" {...args} />,
  args: {
    value: 97.4,
    label: "Success rate",
    format: "percent",
    goalDirection: "higher",
    target: 95,
    comparison: { value: 92.1, label: "vs last week" },
    sparkline: scorecardSparklineUp,
    isLoading: false,
  },
};

export const AtRisk: Story = {
  render: (args) => <ScorecardPanel title="Deploy success rate" {...args} />,
  args: {
    value: 88.6,
    label: "Success rate",
    format: "percent",
    goalDirection: "higher",
    thresholds: scorecardHigherBands,
    comparison: { value: 90.3, label: "vs last week" },
    sparkline: scorecardSparklineDown,
    isLoading: false,
  },
};

export const OffTrack: Story = {
  render: (args) => <ScorecardPanel title="Error rate" {...args} />,
  args: {
    value: 7.8,
    label: "Error rate",
    format: "percent",
    goalDirection: "lower",
    thresholds: scorecardLowerBands,
    comparison: { value: 4.2, label: "vs last week" },
    sparkline: scorecardSparklineUp,
    isLoading: false,
  },
};

export const SimpleTarget: Story = {
  render: (args) => <ScorecardPanel title="Monthly deploys" {...args} />,
  args: {
    value: 132,
    label: "Deploys this month",
    format: "number",
    goalDirection: "higher",
    target: 100,
    comparison: { value: 118, label: "vs last month" },
    isLoading: false,
  },
};

export const MultiBand: Story = {
  render: (args) => <ScorecardPanel title="p95 latency" {...args} />,
  args: {
    value: 240,
    label: "p95 latency",
    format: "number",
    suffix: " ms",
    goalDirection: "lower",
    thresholds: [
      { at: 200, status: "good" },
      { at: 400, status: "warn" },
      { at: 1000, status: "bad" },
    ],
    comparison: { value: 265, label: "vs last week" },
    sparkline: scorecardSparklineDown,
    isLoading: false,
  },
};

export const WithProgressBar: Story = {
  render: (args) => <ScorecardPanel title="Test coverage" height={240} {...args} />,
  args: {
    value: 82,
    label: "Test coverage",
    format: "percent",
    goalDirection: "higher",
    target: 90,
    showProgress: true,
    comparison: { value: 79, label: "vs last release" },
    sparkline: scorecardSparklineUp,
    isLoading: false,
  },
};

export const Neutral: Story = {
  render: (args) => <ScorecardPanel title="Total runs" {...args} />,
  args: {
    value: 4820,
    label: "Total runs",
    format: "number",
    comparison: { value: 4460, label: "vs last week" },
    sparkline: scorecardSparklineUp,
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <ScorecardPanel title="Deploy success rate" {...args} />,
  args: {
    value: null,
    label: "Success rate",
    isLoading: true,
  },
};

export const Empty: Story = {
  render: (args) => <ScorecardPanel title="Deploy success rate" {...args} />,
  args: {
    value: null,
    label: "Success rate",
    isLoading: false,
  },
};
