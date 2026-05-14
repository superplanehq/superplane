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
    content: `The latest run [Health check passed](run:78848cb6-0c52-4c69-8e47-b6631bd703ec|passed) completed in 45s. The previous run [API timeout on node 3](run:2999a5f1-1234-5678-9abc-def012345678|failed) failed due to a timeout.`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const RunsInTable: Story = {
  args: {
    content: `| Run | Duration | Notes |
|-----|----------|-------|
| [Health check OK](run:78848cb6-0c52-4c69-8e47-b6631bd703ec|passed) | 45s | All nodes passed |
| [Timeout on API call](run:2999a5f1-1234-5678-9abc-def012345678|failed) | 0s | Node 3 timed out |
| [Deploy to staging](run:1e8cf8a2-abcd-ef01-2345-678901234567|passed) | 36s | Clean deploy |
| [Waiting for approval](run:366b0a12-1111-2222-3333-444455556666|running) | — | Pending |`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const MixedStatuses: Story = {
  args: {
    content: `Here are your last 5 runs:

- [Morning health check](run:78848cb6-0c52-4c69-8e47-b6631bd703ec|passed) — 45s, all green
- [Deploy v2.1.0](run:2999a5f1-1234-5678-9abc-def012345678|passed) — 36s
- [Nightly backup failed](run:1e8cf8a2-abcd-ef01-2345-678901234567|failed) — timeout at SSH node
- [Manual test run](run:366b0a12-1111-2222-3333-444455556666|failed) — bad expression
- [Rolling deploy](run:e63e35a0-5555-6666-7777-888899990000|running) — in progress`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};
