import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { TimeAgo } from "@/components/TimeAgo";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import { RUN_STATUS_META, buildNodeMap, buildRunPresentation, type RunStatusKey } from "./runPresentation";
import { RunCanvas } from "./storybooks/RunCanvas";
import { AccordionNodeList } from "./storybooks/accordionParts";
import {
  DEPLOY_NODE_ID,
  RUNS_STORY_CANVAS_ID,
  RunsStorySeed,
  mockRuns,
  mockWorkflowNodes,
} from "./storybooks/fixtures";

interface DecoratedRun {
  run: CanvasesCanvasRun;
  title: string;
  triggerName: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
}

function RunTreeRow({
  decoratedRun,
  isExpanded,
  onToggle,
  componentIconMap,
}: {
  decoratedRun: DecoratedRun;
  isExpanded: boolean;
  onToggle: (runId: string) => void;
  componentIconMap: Record<string, string>;
}) {
  const { run, title, triggerName, status, triggerNode } = decoratedRun;
  const iconSrc = getHeaderIconSrc(triggerNode?.component);
  const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;
  const statusMeta = RUN_STATUS_META[status];

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => run.id && onToggle(run.id)}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        if (run.id) onToggle(run.id);
      }}
      className={cn(
        "flex w-full cursor-pointer items-center gap-1.5 px-3 py-2 text-left transition-colors",
        isExpanded ? "bg-sky-100" : "hover:bg-gray-50",
      )}
    >
      <ChevronRight
        className={cn(
          "h-3.5 w-3.5 shrink-0 text-gray-400 transition-transform",
          isExpanded ? "rotate-90 text-gray-700" : "",
        )}
      />
      <RunNodeIcon
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        alt={triggerName}
        size={RUN_NODE_ICON_SIZE}
        className={cn("h-3.5 w-3.5 shrink-0", isExpanded ? "text-gray-800" : "text-gray-500")}
      />
      <span
        aria-label={statusMeta.label}
        title={statusMeta.label}
        className={cn("inline-block h-2 w-2 shrink-0 rounded-full", statusMeta.dotClassName)}
      />
      <span
        className={cn(
          "max-w-[35%] shrink-0 truncate rounded px-1.5 py-0.5 text-[10px] font-medium",
          isExpanded ? "bg-sky-200 text-sky-800" : "bg-slate-100 text-slate-600",
        )}
      >
        {triggerName}
      </span>
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-xs",
          isExpanded ? "font-semibold text-sky-900" : "font-medium text-gray-800",
        )}
      >
        {title}
      </span>
      {run.createdAt ? (
        <span className="shrink-0 text-xs tabular-nums text-gray-500">
          <TimeAgo date={run.createdAt} includeAgo={false} />
        </span>
      ) : null}
    </div>
  );
}

function RunTree({
  canvasId,
  runs,
  workflowNodes,
  expandedRunId,
  expandedNodeId,
  onToggleRun,
  onToggleNode,
}: {
  canvasId: string;
  runs: CanvasesCanvasRun[];
  workflowNodes: ComponentsNode[];
  expandedRunId: string | null;
  expandedNodeId: string | null;
  onToggleRun: (runId: string) => void;
  onToggleNode: (nodeId: string) => void;
}) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const decoratedRuns = useMemo<DecoratedRun[]>(
    () => runs.map((run) => buildRunPresentation(run, nodeMap)),
    [runs, nodeMap],
  );

  return (
    <div className="min-h-0 min-w-0 flex-1 divide-y divide-slate-950/10 overflow-x-hidden overflow-y-auto">
      {decoratedRuns.map((decoratedRun) => {
        const isExpanded = expandedRunId === decoratedRun.run.id;

        return (
          <div key={decoratedRun.run.id}>
            <RunTreeRow
              decoratedRun={decoratedRun}
              isExpanded={isExpanded}
              onToggle={onToggleRun}
              componentIconMap={{}}
            />
            {isExpanded ? (
              <div className="border-l-2 border-sky-100 bg-white pl-3">
                <AccordionNodeList
                  canvasId={canvasId}
                  run={decoratedRun.run}
                  workflowNodes={workflowNodes}
                  expandedNodeId={expandedNodeId}
                  onToggleNode={onToggleNode}
                />
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function CanvasPlaceholder() {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center bg-slate-50">
      <div className="max-w-sm text-center text-sm text-slate-400">Expand a run in the sidebar to inspect it.</div>
    </div>
  );
}

function RunTreePlayground({
  initialRunId = null,
  initialNodeId = null,
}: {
  initialRunId?: string | null;
  initialNodeId?: string | null;
}) {
  const [expandedRunId, setExpandedRunId] = useState<string | null>(initialRunId);
  const [expandedNodeId, setExpandedNodeId] = useState<string | null>(initialNodeId);

  const expandedRun = useMemo(() => mockRuns.find((run) => run.id === expandedRunId) ?? null, [expandedRunId]);

  const toggleRun = (runId: string) =>
    setExpandedRunId((current) => {
      const next = current === runId ? null : runId;
      setExpandedNodeId(null);
      return next;
    });

  const toggleNode = (nodeId: string) => setExpandedNodeId((current) => (current === nodeId ? null : nodeId));

  const selectNode = (nodeId: string) => setExpandedNodeId(nodeId);

  return (
    <div className="flex h-screen min-h-0 bg-white">
      <CanvasRunsSidebar isOpen>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          <div className="flex h-9 shrink-0 items-center border-b border-b-slate-950/10 px-3 text-[13px] font-semibold text-gray-700">
            Runs
          </div>
          <RunTree
            canvasId={RUNS_STORY_CANVAS_ID}
            runs={mockRuns}
            workflowNodes={mockWorkflowNodes}
            expandedRunId={expandedRunId}
            expandedNodeId={expandedNodeId}
            onToggleRun={toggleRun}
            onToggleNode={toggleNode}
          />
        </div>
      </CanvasRunsSidebar>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {expandedRun ? (
          <RunCanvas run={expandedRun} selectedNodeId={expandedNodeId} onSelectNode={selectNode} />
        ) : (
          <CanvasPlaceholder />
        )}
      </div>
    </div>
  );
}

const meta = {
  title: "Runs Proto/Run Tree",
  component: RunTreePlayground,
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
} satisfies Meta<typeof RunTreePlayground>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <RunTreePlayground />,
};

export const RunExpanded: Story = {
  render: () => <RunTreePlayground initialRunId="run-passed" />,
};

export const NodeExpanded: Story = {
  render: () => <RunTreePlayground initialRunId="run-passed" initialNodeId={DEPLOY_NODE_ID} />,
};
