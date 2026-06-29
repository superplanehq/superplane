import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame, memoryEntries, metricRows } from "../__stories__/storyHelpers";
import type { MemoryNumberSource } from "../panelTypes";
import { WidgetNumber } from "./WidgetNumber";

/**
 * Number panel renderer. Aggregates static `rows` via `render.aggregation`, or
 * combines independent memory sources in `composite` mode. The `inline` variant
 * drops outer padding so several values can sit side-by-side in a multi-number
 * panel.
 */
const meta = {
  title: "Console/Number",
  component: WidgetNumber,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
    variant: { control: "inline-radio", options: ["panel", "inline"] },
  },
} satisfies Meta<typeof WidgetNumber>;

export default meta;
type Story = StoryObj<typeof meta>;

function NumberPanel({
  title,
  height = 180,
  ...props
}: { title?: string; height?: number } & ComponentProps<typeof WidgetNumber>) {
  return (
    <PanelFrame height={height}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Metric"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetNumber {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

export const SingleValue: Story = {
  render: (args) => <NumberPanel title="Total runs" {...args} />,
  args: {
    render: { kind: "number", aggregation: "sum", field: "total", label: "Total runs" },
    rows: metricRows,
    isLoading: false,
  },
};

export const WithSparkline: Story = {
  render: (args) => <NumberPanel title="Runs (with trend)" {...args} />,
  args: {
    render: {
      kind: "number",
      aggregation: "sum",
      field: "total",
      label: "Total runs",
      sparklineField: "total",
    },
    rows: metricRows,
    isLoading: false,
  },
};

export const PrefixSuffix: Story = {
  render: (args) => <NumberPanel title="Spend" {...args} />,
  args: {
    render: {
      kind: "number",
      aggregation: "sum",
      field: "total",
      label: "Monthly spend",
      prefix: "$",
      suffix: " /mo",
      format: "number",
    },
    rows: metricRows,
    isLoading: false,
  },
};

const compositeSources: MemoryNumberSource[] = [
  { namespace: "deploys", aggregation: "sum", field: "count" },
  { namespace: "rollbacks", aggregation: "sum", field: "count" },
];

export const Composite: Story = {
  render: (args) => <NumberPanel title="Deploys + rollbacks" {...args} />,
  args: {
    render: { kind: "number", label: "Total memory events" },
    rows: [],
    isLoading: false,
    composite: {
      entries: memoryEntries,
      sources: compositeSources,
      combine: "sum",
    },
  },
};

export const InlineRow: Story = {
  render: () => (
    <PanelFrame height={160}>
      <TypedPanelShell
        title="Multi-metric"
        fallbackTitle="Metrics"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <div className="flex h-full flex-wrap items-center gap-6 p-4">
          <WidgetNumber
            variant="inline"
            render={{ kind: "number", aggregation: "sum", field: "total", label: "Total" }}
            rows={metricRows}
            isLoading={false}
          />
          <WidgetNumber
            variant="inline"
            render={{ kind: "number", aggregation: "avg", field: "passed", label: "Avg passed", format: "number" }}
            rows={metricRows}
            isLoading={false}
          />
          <WidgetNumber
            variant="inline"
            render={{ kind: "number", aggregation: "max", field: "total", label: "Peak" }}
            rows={metricRows}
            isLoading={false}
          />
        </div>
      </TypedPanelShell>
    </PanelFrame>
  ),
};

export const Loading: Story = {
  render: (args) => <NumberPanel title="Total runs" {...args} />,
  args: {
    render: { kind: "number", aggregation: "sum", field: "total", label: "Total runs" },
    rows: [],
    isLoading: true,
  },
};

export const Empty: Story = {
  render: (args) => <NumberPanel title="Total runs" {...args} />,
  args: {
    render: { kind: "number", aggregation: "sum", field: "total", label: "Total runs" },
    rows: [],
    isLoading: false,
  },
};
