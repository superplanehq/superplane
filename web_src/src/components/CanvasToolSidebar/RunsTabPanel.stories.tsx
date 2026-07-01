import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState, type ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { RUNS_STORY_CANVAS_ID, RunsStorySeed, mockRuns, mockWorkflowNodes } from "@/ui/Runs/storybooks/fixtures";
import { RunsTabPanel } from "./RunsTabPanel";

function SidebarFrame({ children }: { children: ReactNode }) {
  return (
    <RunsStorySeed>
      <MemoryRouter>
        <div className="flex h-[42rem] bg-slate-50">
          <CanvasRunsSidebar isOpen>{children}</CanvasRunsSidebar>
        </div>
      </MemoryRouter>
    </RunsStorySeed>
  );
}

const meta = {
  title: "Runs/Runs Sidebar",
  component: RunsTabPanel,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <SidebarFrame>
        <Story />
      </SidebarFrame>
    ),
  ],
} satisfies Meta<typeof RunsTabPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

function RunsSidebarPlayground({ initialRunId = null }: { initialRunId?: string | null }) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(initialRunId);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  return (
    <RunsTabPanel
      canvasId={RUNS_STORY_CANVAS_ID}
      runs={mockRuns}
      workflowNodes={mockWorkflowNodes}
      selectedRunId={selectedRunId}
      selectedNodeId={selectedNodeId}
      initialOpenDetail={!!initialRunId}
      onSelectRun={(runId) => {
        console.log("select run", runId);
        setSelectedRunId(runId);
      }}
      onSelectNode={(nodeId) => {
        console.log("select node", nodeId);
        setSelectedNodeId(nodeId);
      }}
      onSelectLiveCanvas={() => {
        console.log("select live canvas");
        setSelectedRunId(null);
        setSelectedNodeId(null);
      }}
      onBackToRunList={() => {
        console.log("back to run list");
        setSelectedNodeId(null);
      }}
    />
  );
}

export const RunList: Story = {
  render: () => <RunsSidebarPlayground />,
};

export const RunDetail: Story = {
  render: () => <RunsSidebarPlayground initialRunId="run-passed" />,
};

export const Loading: Story = {
  render: () => (
    <RunsTabPanel
      canvasId={RUNS_STORY_CANVAS_ID}
      runs={[]}
      workflowNodes={mockWorkflowNodes}
      selectedRunId={null}
      onSelectRun={(runId) => console.log("select run", runId)}
      onSelectLiveCanvas={() => console.log("select live canvas")}
      isLoading
    />
  ),
};

export const ErrorState: Story = {
  render: () => (
    <RunsTabPanel
      canvasId={RUNS_STORY_CANVAS_ID}
      runs={[]}
      workflowNodes={mockWorkflowNodes}
      selectedRunId={null}
      onSelectRun={(runId) => console.log("select run", runId)}
      onSelectLiveCanvas={() => console.log("select live canvas")}
      isError
      onRetry={() => console.log("retry")}
    />
  ),
};

export const Empty: Story = {
  render: () => (
    <RunsTabPanel
      canvasId={RUNS_STORY_CANVAS_ID}
      runs={[]}
      workflowNodes={mockWorkflowNodes}
      selectedRunId={null}
      onSelectRun={(runId) => console.log("select run", runId)}
      onSelectLiveCanvas={() => console.log("select live canvas")}
    />
  ),
};
