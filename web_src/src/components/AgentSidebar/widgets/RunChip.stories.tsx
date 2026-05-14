import type { Meta, StoryObj } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { RichMessage } from "./RichMessage";

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/RunChips",
  component: RichMessage,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="max-w-md bg-slate-100 rounded-lg p-4">
          <div className="bg-slate-100 rounded-lg px-3 py-2 text-sm text-slate-900">
            <Story />
          </div>
        </div>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof RichMessage>;

export const InlineRunReference: Story = {
  args: {
    content: `The latest run [#78848c](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) passed in 45s. The previous run [#2999a5](run:2999a5f1-1234-5678-9abc-def012345678) failed due to a timeout.`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const RunsInTable: Story = {
  args: {
    content: `| Run | Status | Duration |
|-----|--------|----------|
| [#78848c](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) | ✅ Passed | 45s |
| [#2999a5](run:2999a5f1-1234-5678-9abc-def012345678) | ❌ Failed | 0s |
| [#1e8cf8](run:1e8cf8a2-abcd-ef01-2345-678901234567) | ✅ Passed | 36s |`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const RunWithChart: Story = {
  args: {
    content: `Run [#78848c](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) took 45s — here's how that compares:

:::chart
type: bar
title: Recent Run Durations (seconds)
x: ["#78848c", "#2999a5", "#1e8cf8", "#366b0a"]
series:
  - name: Duration
    data: [45, 0, 36, 95]
    color: "#8b5cf6"
:::`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};
