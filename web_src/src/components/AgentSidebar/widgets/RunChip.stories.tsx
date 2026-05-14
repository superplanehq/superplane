import type { Meta, StoryObj } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RichMessage } from "./RichMessage";

const queryClient = new QueryClient();

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/RunChips",
  component: RichMessage,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <div className="max-w-md bg-slate-100 rounded-lg p-4">
            <div className="bg-slate-100 rounded-lg px-3 py-2 text-sm text-slate-900">
              <Story />
            </div>
          </div>
        </MemoryRouter>
      </QueryClientProvider>
    ),
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

export const RunsInList: Story = {
  args: {
    content: `Here are your last 5 runs:

- [run](run:78848cb6-0c52-4c69-8e47-b6631bd703ec) — 45s, all green
- [run](run:2999a5f1-1234-5678-9abc-def012345678) — 36s
- [run](run:1e8cf8a2-abcd-ef01-2345-678901234567) — timeout at SSH node
- [run](run:366b0a12-1111-2222-3333-444455556666) — bad expression
- [run](run:e63e35a0-5555-6666-7777-888899990000) — in progress`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};
