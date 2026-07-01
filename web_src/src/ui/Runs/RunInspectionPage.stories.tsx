import type { Meta, StoryObj } from "@storybook/react-vite";
import { useMemo, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { RunsTabPanel } from "@/components/CanvasToolSidebar/RunsTabPanel";
import { RunNodeDetailPane } from "./RunNodeDetailPane";
import { RunCanvas } from "./storybooks/RunCanvas";
import {
  DEPLOY_NODE_ID,
  RUNS_STORY_CANVAS_ID,
  RunsStorySeed,
  mockRuns,
  mockWorkflowNodes,
} from "./storybooks/fixtures";

function CanvasPlaceholder() {
  return (
    <div className="relative flex min-h-0 flex-1 items-center justify-center bg-slate-50">
      <div className="max-w-sm text-center text-sm text-slate-400">
        Canvas preview. Open a run in the sidebar to see its nodes here.
      </div>
    </div>
  );
}

function RunInspectionPagePlayground({ initialNodeId = null }: { initialNodeId?: string | null }) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>("run-passed");
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(initialNodeId);

  const selectedRun = useMemo(() => mockRuns.find((run) => run.id === selectedRunId) ?? null, [selectedRunId]);

  return (
    <div className="flex h-screen min-h-0 bg-white">
      <CanvasRunsSidebar isOpen>
        <RunsTabPanel
          canvasId={RUNS_STORY_CANVAS_ID}
          runs={mockRuns}
          workflowNodes={mockWorkflowNodes}
          selectedRunId={selectedRunId}
          selectedNodeId={selectedNodeId}
          initialOpenDetail={!!selectedRunId}
          onSelectRun={(runId) => {
            console.log("select run", runId);
            setSelectedRunId(runId);
            setSelectedNodeId(null);
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
      </CanvasRunsSidebar>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {selectedRun ? (
          <RunCanvas run={selectedRun} selectedNodeId={selectedNodeId} onSelectNode={setSelectedNodeId} />
        ) : (
          <CanvasPlaceholder />
        )}

        {selectedRun && selectedNodeId ? (
          <RunNodeDetailPane
            canvasId={RUNS_STORY_CANVAS_ID}
            run={selectedRun}
            nodeId={selectedNodeId}
            workflowNodes={mockWorkflowNodes}
            onClose={() => {
              console.log("close node detail pane");
              setSelectedNodeId(null);
            }}
            onNavigateNode={(nodeId) => {
              console.log("navigate node", nodeId);
              setSelectedNodeId(nodeId);
            }}
          />
        ) : null}
      </div>
    </div>
  );
}

const meta = {
  title: "Runs Proto/Run Inspection Page (Bottom Pane)",
  component: RunInspectionPagePlayground,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <RunsStorySeed>
        <MemoryRouter>
          <Story />
        </MemoryRouter>
      </RunsStorySeed>
    ),
  ],
} satisfies Meta<typeof RunInspectionPagePlayground>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <RunInspectionPagePlayground />,
};

export const NodePreselected: Story = {
  render: () => <RunInspectionPagePlayground initialNodeId={DEPLOY_NODE_ID} />,
};
