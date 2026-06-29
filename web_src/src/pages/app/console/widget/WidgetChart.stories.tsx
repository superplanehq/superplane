import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame, serviceRows, timeseriesRows } from "../__stories__/storyHelpers";
import { WidgetChart } from "./WidgetChart";

/**
 * Chart panel renderer (Recharts-backed). One story per chart `type` plus
 * multi-series, loading, and empty states. Rows are static; the renderer bins
 * and formats them based on the `render` config.
 */
const meta = {
  title: "Console/Chart",
  component: WidgetChart,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
  },
} satisfies Meta<typeof WidgetChart>;

export default meta;
type Story = StoryObj<typeof meta>;

function ChartPanel({ title, ...props }: { title?: string } & ComponentProps<typeof WidgetChart>) {
  return (
    <PanelFrame>
      <TypedPanelShell
        title={title}
        fallbackTitle="Chart"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetChart {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

export const Bar: Story = {
  render: (args) => <ChartPanel title="Errors by service" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "errors", label: "Errors" }],
      yLabel: "Errors",
    },
    rows: serviceRows,
    isLoading: false,
  },
};

export const StackedBar: Story = {
  render: (args) => <ChartPanel title="Runs by day" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "stacked-bar",
      xField: "day",
      series: [
        { field: "passed", label: "Passed" },
        { field: "failed", label: "Failed" },
      ],
    },
    rows: timeseriesRows,
    isLoading: false,
  },
};

export const StackedBarPivoted: Story = {
  render: (args) => <ChartPanel title="Daily token cost" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "stacked-bar",
      xField: "day",
      seriesField: "model",
      series: [{ field: "cost", label: "Cost", prefix: "$" }],
      yLabel: "USD",
    },
    rows: [
      { day: "May 01", model: "Claude Haiku 4.5", cost: 120 },
      { day: "May 01", model: "Claude Sonnet 4.6", cost: 340 },
      { day: "May 01", model: "Claude Opus 4.6", cost: 890 },
      { day: "May 02", model: "Claude Haiku 4.5", cost: 95 },
      { day: "May 02", model: "Claude Sonnet 4.6", cost: 410 },
      { day: "May 02", model: "Claude Opus 4.6", cost: 1200 },
      { day: "May 02", model: "Claude Opus 4.7", cost: 220 },
    ],
    isLoading: false,
  },
};

export const Line: Story = {
  render: (args) => <ChartPanel title="Passing runs trend" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "line",
      xField: "day",
      series: [{ field: "passed", label: "Passed" }],
    },
    rows: timeseriesRows,
    isLoading: false,
  },
};

export const Area: Story = {
  render: (args) => <ChartPanel title="Passing runs (area)" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "area",
      xField: "day",
      series: [{ field: "passed", label: "Passed" }],
    },
    rows: timeseriesRows,
    isLoading: false,
  },
};

export const Donut: Story = {
  render: (args) => <ChartPanel title="Requests by service" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "donut",
      xField: "service",
      series: [{ field: "requests", label: "Requests" }],
    },
    rows: serviceRows,
    isLoading: false,
  },
};

export const MultiSeries: Story = {
  render: (args) => <ChartPanel title="Passed vs failed" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "line",
      xField: "day",
      series: [
        { field: "passed", label: "Passed" },
        { field: "failed", label: "Failed" },
      ],
      legend: "show",
    },
    rows: timeseriesRows,
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <ChartPanel title="Errors by service" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "errors", label: "Errors" }],
    },
    rows: [],
    isLoading: true,
  },
};

export const Empty: Story = {
  render: (args) => <ChartPanel title="Errors by service" {...args} />,
  args: {
    render: {
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "errors", label: "Errors" }],
    },
    rows: [],
    isLoading: false,
  },
};
