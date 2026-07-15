import type { Meta, StoryObj } from "@storybook/react";

import { RUN_STATUS_META, type RunStatusKey } from "./runPresentation";
import { RunStatusBadge } from "./RunStatusBadge";

const ALL_STATUSES = Object.keys(RUN_STATUS_META) as RunStatusKey[];

const meta = {
  title: "Canvas/RunStatusBadge",
  component: RunStatusBadge,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    status: {
      control: { type: "select" },
      options: ALL_STATUSES,
    },
  },
  args: {
    status: "running",
  },
} satisfies Meta<typeof RunStatusBadge>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Running: Story = {
  args: { status: "running" },
};

export const Passed: Story = {
  args: { status: "passed" },
};

export const Failed: Story = {
  args: { status: "failed" },
};

export const Cancelled: Story = {
  args: { status: "cancelled" },
};

export const Unknown: Story = {
  args: { status: "unknown" },
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-white p-4">
        {ALL_STATUSES.map((status) => (
          <RunStatusBadge key={`light-${status}`} status={status} />
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border border-gray-700 bg-gray-950 p-4">
        {ALL_STATUSES.map((status) => (
          <RunStatusBadge key={`dark-${status}`} status={status} />
        ))}
      </div>
    </div>
  ),
};
