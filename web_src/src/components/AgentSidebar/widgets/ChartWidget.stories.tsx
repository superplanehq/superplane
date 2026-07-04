import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChartWidget } from "./ChartWidget";

const meta: Meta<typeof ChartWidget> = {
  title: "AgentSidebar/Charts",
  component: ChartWidget,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <div className="max-w-md bg-white border border-slate-200 rounded-lg p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof ChartWidget>;

export const RunSuccessRate: Story = {
  args: {
    config: {
      type: "line",
      title: "Run Success Rate (Last 7 Days)",
      x: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
      series: [
        { name: "Successful", data: [12, 15, 11, 14, 13, 8, 10], color: "#22c55e" },
        { name: "Failed", data: [2, 1, 3, 0, 2, 1, 0], color: "#ef4444" },
      ],
    },
  },
};

export const ExecutionsPerNode: Story = {
  args: {
    config: {
      type: "bar",
      title: "Executions Per Node (Last 24h)",
      x: ["Webhook", "API Call", "Check Status", "Notify OK", "Notify Fail"],
      series: [{ name: "Executions", data: [48, 48, 48, 41, 7], color: "#8b5cf6" }],
    },
  },
};

export const RunOutcomes: Story = {
  args: {
    config: {
      type: "pie",
      title: "Run Outcomes (Last 30 Days)",
      data: [
        { name: "Success", value: 312, color: "#22c55e" },
        { name: "Failed", value: 28, color: "#ef4444" },
        { name: "Timed Out", value: 8, color: "#f59e0b" },
        { name: "Cancelled", value: 3, color: "#94a3b8" },
      ],
    },
  },
};

export const LatencyOverTime: Story = {
  args: {
    config: {
      type: "area",
      title: "Avg Execution Latency (ms)",
      x: ["00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "24:00"],
      series: [
        { name: "P50", data: [120, 115, 180, 220, 195, 160, 130], color: "#8b5cf6" },
        { name: "P95", data: [450, 420, 680, 890, 750, 520, 480], color: "#f59e0b" },
        { name: "P99", data: [1200, 980, 1500, 2100, 1800, 1100, 950], color: "#ef4444" },
      ],
    },
  },
};

export const WeeklyRunVolume: Story = {
  args: {
    config: {
      type: "bar",
      title: "Weekly Run Volume (4 Weeks)",
      x: ["Week 1", "Week 2", "Week 3", "Week 4"],
      series: [
        { name: "Success", data: [72, 85, 81, 90], color: "#22c55e" },
        { name: "Failed", data: [6, 7, 7, 3], color: "#ef4444" },
      ],
    },
  },
};

export const NodeFailureRate: Story = {
  args: {
    config: {
      type: "bar",
      title: "Node Failure Rate (%)",
      x: ["SSH Deploy", "API Health", "Slack Notify", "DB Backup", "DNS Update"],
      series: [{ name: "Failure %", data: [12.5, 4.2, 1.1, 8.7, 0.3], color: "#ef4444" }],
    },
  },
};

export const MultiSeriesLine: Story = {
  args: {
    config: {
      type: "line",
      title: "Canvas Activity (Last 14 Days)",
      x: ["D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10", "D11", "D12", "D13", "D14"],
      series: [
        { name: "Triggers", data: [24, 31, 28, 35, 42, 38, 22, 19, 33, 41, 45, 39, 37, 44], color: "#8b5cf6" },
        { name: "Actions", data: [72, 93, 84, 105, 126, 114, 66, 57, 99, 123, 135, 117, 111, 132], color: "#06b6d4" },
        { name: "Failures", data: [3, 2, 5, 1, 4, 2, 1, 0, 3, 2, 6, 1, 2, 3], color: "#ef4444" },
      ],
    },
  },
};
