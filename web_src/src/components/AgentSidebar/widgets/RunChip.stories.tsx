import type { Meta, StoryObj } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RichMessage } from "./RichMessage";

const MOCK_RUNS = [
  {
    id: "78848cb6-0c52-4c69-8e47-b6631bd703ec",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-14T19:08:57Z",
    finishedAt: "2026-05-14T19:09:42Z",
    executions: [
      { nodeId: "manual-run-1234", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "random-wait-5678", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "call-api-9abc", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "check-result-def0", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "notify-success-1111", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    ],
  },
  {
    id: "2999a5f1-1234-5678-9abc-def012345678",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    createdAt: "2026-05-14T18:55:10Z",
    finishedAt: "2026-05-14T18:55:10Z",
    executions: [
      { nodeId: "manual-run-1234", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "random-wait-5678", state: "STATE_FINISHED", result: "RESULT_FAILED" },
    ],
  },
  {
    id: "1e8cf8a2-abcd-ef01-2345-678901234567",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-14T18:30:00Z",
    finishedAt: "2026-05-14T18:30:36Z",
    executions: [
      { nodeId: "manual-run-1234", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "call-api-9abc", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "notify-success-1111", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    ],
  },
  {
    id: "366b0a12-1111-2222-3333-444455556666",
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    createdAt: "2026-05-14T20:30:00Z",
    executions: [
      { nodeId: "manual-run-1234", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      { nodeId: "random-wait-5678", state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
    ],
  },
  {
    id: "e63e35a0-5555-6666-7777-888899990000",
    state: "STATE_FINISHED",
    result: "RESULT_CANCELLED",
    createdAt: "2026-05-14T17:00:00Z",
    finishedAt: "2026-05-14T17:00:12Z",
    executions: [
      { nodeId: "manual-run-1234", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    ],
  },
];

function createSeededQueryClient() {
  const qc = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  qc.setQueryData(["canvas", "runs"], {
    pages: [{ runs: MOCK_RUNS }],
  });
  return qc;
}

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/RunChips",
  component: RichMessage,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => {
      const qc = createSeededQueryClient();
      return (
        <QueryClientProvider client={qc}>
          <MemoryRouter>
            <div className="max-w-md bg-slate-100 rounded-lg p-4">
              <div className="bg-slate-100 rounded-lg px-3 py-2 text-sm text-slate-900">
                <Story />
              </div>
            </div>
          </MemoryRouter>
        </QueryClientProvider>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof RichMessage>;

export const InlineRunReference: Story = {
  args: {
    content: `The latest run [run](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) completed in 45s. The previous run [run](run:2999a5f1-1234-5678-9abc-def012345678) failed due to a timeout.`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const AllStatuses: Story = {
  args: {
    content: `Run status examples:

- Passed: [run](run:78848cb6-0c52-4c69-8e47-b6631bd703ec)
- Failed: [run](run:2999a5f1-1234-5678-9abc-def012345678)
- Running: [run](run:366b0a12-1111-2222-3333-444455556666)
- Cancelled: [run](run:e63e35a0-5555-6666-7777-888899990000)`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const RunsInTable: Story = {
  args: {
    content: `| Run | Duration | Notes |
|-----|----------|-------|
| [run](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) | 45s | All nodes passed |
| [run](run:2999a5f1-1234-5678-9abc-def012345678) | 0s | Node 3 timed out |
| [run](run:1e8cf8a2-abcd-ef01-2345-678901234567) | 36s | Clean deploy |
| [run](run:366b0a12-1111-2222-3333-444455556666) | — | In progress |
| [run](run:e63e35a0-5555-6666-7777-888899990000) | 12s | User cancelled |`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};
