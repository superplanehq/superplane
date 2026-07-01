import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { ResizableBottomPane } from "@/ui/CanvasPage/ResizableBottomPane";
import {
  DEPLOY_NODE_ID,
  NOTIFY_NODE_ID,
  TRIGGER_NODE_ID,
  mockFailedRun,
  mockPassedExecutions,
  mockPassedRun,
  mockWorkflowNodes,
} from "./storybooks/fixtures";
import { RunNodeDetailContent } from "./RunNodeDetailContent";

function BottomPaneFrame({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <div className="flex h-screen flex-col justify-end bg-slate-100">
        <ResizableBottomPane defaultHeight={340}>{children}</ResizableBottomPane>
      </div>
    </MemoryRouter>
  );
}

const meta = {
  title: "Runs/Node Detail Pane (Bottom)",
  component: RunNodeDetailContent,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <BottomPaneFrame>
        <Story />
      </BottomPaneFrame>
    ),
  ],
} satisfies Meta<typeof RunNodeDetailContent>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    run: mockPassedRun,
    nodeId: DEPLOY_NODE_ID,
    workflowNodes: mockWorkflowNodes,
    executions: mockPassedExecutions,
    onClose: () => console.log("close"),
    onNavigateNode: (nodeId) => console.log("navigate node", nodeId),
  },
};

export const TriggerNode: Story = {
  args: {
    run: mockPassedRun,
    nodeId: TRIGGER_NODE_ID,
    workflowNodes: mockWorkflowNodes,
    executions: mockPassedExecutions,
    onClose: () => console.log("close"),
    onNavigateNode: (nodeId) => console.log("navigate node", nodeId),
  },
};

export const Loading: Story = {
  args: {
    run: mockPassedRun,
    nodeId: NOTIFY_NODE_ID,
    workflowNodes: mockWorkflowNodes,
    executions: [],
    isExecutionsLoading: true,
    onClose: () => console.log("close"),
    onNavigateNode: (nodeId) => console.log("navigate node", nodeId),
  },
};

export const NoExecutionData: Story = {
  args: {
    run: mockFailedRun,
    nodeId: DEPLOY_NODE_ID,
    workflowNodes: mockWorkflowNodes,
    executions: [],
    onClose: () => console.log("close"),
    onNavigateNode: (nodeId) => console.log("navigate node", nodeId),
  },
};
