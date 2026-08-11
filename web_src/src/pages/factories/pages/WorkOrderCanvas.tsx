import { useCallback, useMemo, useState } from "react";
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  addEdge,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
  type NodeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

type StepStatus = "passed" | "created" | "success" | "running" | "failed" | "pending";

type StepDetail = {
  startedAt?: string;
  finishedAt?: string;
  duration?: string;
  agent?: string;
  attempt?: string;
  output?: string;
  logs?: string[];
  inputs?: string[];
};

type StepNodeData = {
  title: string;
  status: StepStatus;
  kind: "step" | "action";
  detail?: string;
  meta?: string[];
  showLogs?: boolean;
  details: StepDetail;
};

function statusBadgeClass(status: StepStatus) {
  switch (status) {
    case "passed":
    case "created":
    case "success":
      return "border-[#15803d] bg-[#dcfce7] text-[#166534]";
    case "running":
      return "border-[#2563eb] bg-[#dbeafe] text-[#1d4ed8]";
    case "failed":
      return "border-[#b91c1c] bg-[#fee2e2] text-[#991b1b]";
    case "pending":
      return "border-[#737373] bg-[#f3f3f3] text-[#525252]";
  }
}

function statusLabel(status: StepStatus) {
  switch (status) {
    case "passed":
      return "PASSED";
    case "created":
      return "CREATED";
    case "success":
      return "SUCCESS";
    case "running":
      return "RUNNING";
    case "failed":
      return "FAILED";
    case "pending":
      return "PENDING";
  }
}

function statusSentence(status: StepStatus) {
  switch (status) {
    case "passed":
      return "Completed successfully.";
    case "created":
      return "Resource created and ready.";
    case "success":
      return "Finished with success.";
    case "running":
      return "Currently executing.";
    case "failed":
      return "Failed and stopped.";
    case "pending":
      return "Waiting to start.";
  }
}

function StepNode({ data, selected }: NodeProps<Node<StepNodeData>>) {
  return (
    <div
      className={cn(
        "w-[300px] cursor-pointer rounded-lg border-2 bg-white px-4 py-3 shadow-[0_1px_2px_rgba(38,37,30,0.08)]",
        data.status === "running" && "border-[#2563eb] bg-[#eff6ff]",
        data.status === "pending" && "border-[#a3a3a3] bg-[#f7f7f7]",
        data.status !== "running" && data.status !== "pending" && "border-[#26251e]",
        selected && "ring-2 ring-[#2563eb] ring-offset-2",
      )}
    >
      <Handle type="target" position={Position.Top} className="!size-2.5 !border-2 !border-[#525252] !bg-white" />
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 text-[14px] font-semibold tracking-[-0.01em] text-foreground">{data.title}</div>
        <span
          className={cn(
            "shrink-0 rounded border px-1.5 py-0.5 text-[10px] font-semibold tracking-[0.04em]",
            statusBadgeClass(data.status),
          )}
        >
          {statusLabel(data.status)}
        </span>
      </div>
      {data.detail ? <p className="mt-2 truncate font-mono text-[11px] text-[#525252]">{data.detail}</p> : null}
      {data.meta?.length ? (
        <div className="mt-2 space-y-1">
          {data.meta.map((line) => (
            <p key={line} className="text-[12px] leading-snug text-[#525252]">
              {line}
            </p>
          ))}
        </div>
      ) : null}
      {data.showLogs ? <div className="mt-3 text-[12px] font-medium text-[#2563eb]">View logs</div> : null}
      <Handle
        id="out"
        type="source"
        position={Position.Bottom}
        className="!size-2.5 !border-2 !border-[#525252] !bg-white"
      />
      <Handle
        id="side"
        type="source"
        position={Position.Right}
        className="!top-1/2 !size-2.5 !border-2 !border-[#525252] !bg-white"
      />
    </div>
  );
}

function ActionNode({ data, selected }: NodeProps<Node<StepNodeData>>) {
  return (
    <div
      className={cn(
        "w-[220px] cursor-pointer rounded-md border-2 border-[#737373] bg-[#f7f7f7] px-3 py-2.5",
        selected && "ring-2 ring-[#2563eb] ring-offset-2",
      )}
    >
      <Handle type="target" position={Position.Left} className="!size-2 !border-2 !border-[#525252] !bg-white" />
      <div className="truncate font-mono text-[11px] tracking-[-0.01em] text-foreground">{data.title}</div>
      <div className="mt-1 text-[10px] font-medium tracking-[0.04em] text-muted-foreground">
        {statusLabel(data.status)}
      </div>
    </div>
  );
}

const nodeTypes = {
  step: StepNode,
  action: ActionNode,
};

const MAIN_X = 120;
const SIDE_X = 520;
const EDGE_STYLE = { stroke: "#94a3b8", strokeWidth: 1.5 };

function step(
  id: string,
  position: { x: number; y: number },
  data: Omit<StepNodeData, "kind">,
  editable = false,
): Node<StepNodeData> {
  return {
    id,
    type: "step",
    position,
    data: { ...data, kind: "step" },
    draggable: editable,
  };
}

function action(
  id: string,
  position: { x: number; y: number },
  data: Omit<StepNodeData, "kind" | "showLogs" | "meta">,
  editable = false,
): Node<StepNodeData> {
  return {
    id,
    type: "action",
    position,
    data: { ...data, kind: "action" },
    draggable: editable,
  };
}

function buildWorkflowGraph(editable = false): { nodes: Node<StepNodeData>[]; edges: Edge[] } {
  const nodes: Node<StepNodeData>[] = [
    step(
      "create-branch",
      { x: MAIN_X, y: 0 },
      {
        title: "Create Branch",
        status: "passed",
        detail: "feat/invite-poc · a3f91c2",
        showLogs: true,
        details: {
          startedAt: "14:02:11",
          finishedAt: "14:02:18",
          duration: "7s",
          agent: "git-worker",
          attempt: "1 / 1",
          output: "Branch feat/invite-poc created from main.",
          logs: ["Cloned repository", "Checked out main", "Created feat/invite-poc", "Pushed origin"],
          inputs: ["base: main", "prefix: feat/invite-poc"],
        },
      },
    ),
    step(
      "open-draft-pr",
      { x: MAIN_X, y: 170 },
      {
        title: "Open Draft PR",
        status: "created",
        detail: "PR #148 · Draft",
        details: {
          startedAt: "14:02:19",
          finishedAt: "14:02:24",
          duration: "5s",
          agent: "github-worker",
          attempt: "1 / 1",
          output: "Opened draft PR #148 against main.",
          logs: ["Authenticated GitHub app", "Created draft pull request", "Linked work order"],
          inputs: ["head: feat/invite-poc", "base: main", "draft: true"],
        },
      },
    ),
    step(
      "write-plan",
      { x: MAIN_X, y: 340 },
      {
        title: "Write Plan",
        status: "passed",
        detail: "plan.md · 12 steps",
        showLogs: true,
        details: {
          startedAt: "14:02:25",
          finishedAt: "14:03:41",
          duration: "1m 16s",
          agent: "planner",
          attempt: "1 / 1",
          output: "Wrote plan.md with 12 implementation steps.",
          logs: ["Read repo context", "Drafted plan", "Committed plan.md", "Posted plan comment"],
          inputs: ["goal: Bind Console widget to latest run", "repo: superplanehq/superplane"],
        },
      },
    ),
    action(
      "action-comment",
      { x: SIDE_X, y: 300 },
      {
        title: "github.createIssueComment",
        status: "passed",
        detail: "comment · #8841",
        details: {
          startedAt: "14:03:42",
          finishedAt: "14:03:43",
          duration: "1s",
          agent: "github-worker",
          attempt: "1 / 1",
          output: "Posted plan summary on issue #8841.",
          logs: ["Rendered markdown comment", "Posted via Issues API"],
          inputs: ["issue: #8841"],
        },
      },
    ),
    action(
      "action-notify-start",
      { x: SIDE_X, y: 350 },
      {
        title: "Notify work started",
        status: "passed",
        detail: "slack · #factory",
        details: {
          startedAt: "14:03:43",
          finishedAt: "14:03:44",
          duration: "1s",
          agent: "notifier",
          attempt: "1 / 1",
          output: "Sent start notification to #factory.",
          logs: ["Resolved channel", "Posted message"],
          inputs: ["channel: #factory"],
        },
      },
    ),
    action(
      "action-label",
      { x: SIDE_X, y: 400 },
      {
        title: "github.addIssueLabel",
        status: "passed",
        detail: "label · in-progress",
        details: {
          startedAt: "14:03:44",
          finishedAt: "14:03:45",
          duration: "1s",
          agent: "github-worker",
          attempt: "1 / 1",
          output: "Added in-progress label.",
          logs: ["Applied label in-progress"],
          inputs: ["issue: #8841", "label: in-progress"],
        },
      },
    ),
    action(
      "action-correlate",
      { x: SIDE_X, y: 450 },
      {
        title: "Correlate PR Memory",
        status: "passed",
        detail: "memory · pr-148",
        details: {
          startedAt: "14:03:45",
          finishedAt: "14:03:46",
          duration: "1s",
          agent: "memory",
          attempt: "1 / 1",
          output: "Linked PR #148 to work-order memory.",
          logs: ["Wrote correlation record"],
          inputs: ["pr: 148", "workOrder: poc-4"],
        },
      },
    ),
    step(
      "implementation",
      { x: MAIN_X, y: 560 },
      {
        title: "Implementation",
        status: "running",
        detail: "commits · 4 files",
        showLogs: true,
        details: {
          startedAt: "14:03:47",
          duration: "9m so far",
          agent: "implementor",
          attempt: "1 / 3",
          output: "Editing auth mock routes and tests…",
          logs: [
            "Checked out feat/invite-poc",
            "Updated src/auth/mock.ts",
            "Updated src/auth/session.ts",
            "Running unit tests…",
          ],
          inputs: ["plan.md", "acceptance: mock auth path"],
        },
      },
    ),
    action(
      "action-mark-plan",
      { x: SIDE_X, y: 590 },
      {
        title: "Mark Plan Done",
        status: "pending",
        detail: "waits on implementation",
        details: {
          agent: "orchestrator",
          attempt: "—",
          output: "Queued until Implementation passes.",
          inputs: ["dependsOn: Implementation"],
        },
      },
    ),
    step(
      "babysit",
      { x: MAIN_X, y: 780 },
      {
        title: "Babysit",
        status: "pending",
        meta: ["Software Factory", "Start Babysitting", "Timeout: 1h"],
        details: {
          agent: "software-factory",
          attempt: "—",
          duration: "Timeout: 1h",
          output: "Will start after Implementation finishes.",
          inputs: ["mode: Start Babysitting", "timeout: 1h"],
        },
      },
    ),
    action(
      "action-mark-impl",
      { x: SIDE_X, y: 760 },
      {
        title: "Mark Implementation Done",
        status: "pending",
        detail: "on babysit passed",
        details: {
          agent: "orchestrator",
          attempt: "—",
          output: "Runs when Babysit reports passed.",
          inputs: ["dependsOn: Babysit / passed"],
        },
      },
    ),
    action(
      "action-notify-fail",
      { x: SIDE_X, y: 810 },
      {
        title: "Notify work failed",
        status: "pending",
        detail: "on babysit failed",
        details: {
          agent: "notifier",
          attempt: "—",
          output: "Runs when Babysit reports failed.",
          inputs: ["dependsOn: Babysit / failed", "channel: #factory"],
        },
      },
    ),
    action(
      "action-mark-fail-memory",
      { x: SIDE_X, y: 860 },
      {
        title: "Mark Implementation Failed Memory",
        status: "pending",
        detail: "on babysit failed",
        details: {
          agent: "memory",
          attempt: "—",
          output: "Writes failure memory when Babysit fails.",
          inputs: ["dependsOn: Babysit / failed"],
        },
      },
    ),
  ];

  const edges: Edge[] = [
    {
      id: "e-create-pr",
      source: "create-branch",
      sourceHandle: "out",
      target: "open-draft-pr",
      type: "smoothstep",
      label: "passed",
      style: EDGE_STYLE,
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#94a3b8" },
      labelStyle: { fill: "#737373", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-pr-plan",
      source: "open-draft-pr",
      sourceHandle: "out",
      target: "write-plan",
      type: "smoothstep",
      label: "passed",
      style: EDGE_STYLE,
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#94a3b8" },
      labelStyle: { fill: "#737373", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-plan-comment",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-comment",
      type: "smoothstep",
      style: EDGE_STYLE,
    },
    {
      id: "e-plan-notify",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-notify-start",
      type: "smoothstep",
      style: EDGE_STYLE,
    },
    {
      id: "e-plan-label",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-label",
      type: "smoothstep",
      style: EDGE_STYLE,
    },
    {
      id: "e-plan-correlate",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-correlate",
      type: "smoothstep",
      style: EDGE_STYLE,
    },
    {
      id: "e-plan-impl",
      source: "write-plan",
      sourceHandle: "out",
      target: "implementation",
      type: "smoothstep",
      label: "passed",
      style: EDGE_STYLE,
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#94a3b8" },
      labelStyle: { fill: "#737373", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-impl-mark-plan",
      source: "implementation",
      sourceHandle: "side",
      target: "action-mark-plan",
      type: "smoothstep",
      style: EDGE_STYLE,
    },
    {
      id: "e-impl-babysit",
      source: "implementation",
      sourceHandle: "out",
      target: "babysit",
      type: "smoothstep",
      label: "passed",
      animated: true,
      style: { ...EDGE_STYLE, stroke: "#3b82f6" },
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#3b82f6" },
      labelStyle: { fill: "#737373", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-babysit-mark",
      source: "babysit",
      sourceHandle: "side",
      target: "action-mark-impl",
      type: "smoothstep",
      label: "passed",
      style: EDGE_STYLE,
      labelStyle: { fill: "#737373", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-babysit-notify-fail",
      source: "babysit",
      sourceHandle: "side",
      target: "action-notify-fail",
      type: "smoothstep",
      label: "failed",
      style: { ...EDGE_STYLE, stroke: "#f87171" },
      labelStyle: { fill: "#b91c1c", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
    {
      id: "e-babysit-fail-memory",
      source: "babysit",
      sourceHandle: "side",
      target: "action-mark-fail-memory",
      type: "smoothstep",
      label: "failed",
      style: { ...EDGE_STYLE, stroke: "#f87171" },
      labelStyle: { fill: "#b91c1c", fontSize: 11 },
      labelBgStyle: { fill: "#ffffff" },
      labelBgPadding: [4, 6] as [number, number],
    },
  ];

  return {
    nodes: nodes.map((node) => ({ ...node, draggable: editable })),
    edges,
  };
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[96px_minmax(0,1fr)] gap-2 text-[13px]">
      <div className="text-muted-foreground">{label}</div>
      <div className="min-w-0 break-words text-foreground">{value}</div>
    </div>
  );
}

function StepDetailSidebar({ node, onClose }: { node: Node<StepNodeData>; onClose: () => void }) {
  const { data } = node;
  const { details } = data;

  return (
    <aside className="flex h-full w-[320px] shrink-0 flex-col border-l border-border bg-[#f7f7f7]">
      <div className="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
        <div className="min-w-0">
          <div className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">
            {data.kind === "action" ? "Action" : "Step"}
          </div>
          <h3 className="mt-1 text-[14px] font-semibold tracking-[-0.01em] text-foreground">{data.title}</h3>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Close step details"
        >
          <X className="size-4" strokeWidth={1.75} />
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "rounded border px-1.5 py-0.5 text-[10px] font-semibold tracking-[0.04em]",
              statusBadgeClass(data.status),
            )}
          >
            {statusLabel(data.status)}
          </span>
          <span className="text-[12px] text-muted-foreground">{statusSentence(data.status)}</span>
        </div>

        <div className="mt-5 space-y-3">
          {details.agent ? <DetailRow label="Agent" value={details.agent} /> : null}
          {details.attempt ? <DetailRow label="Attempt" value={details.attempt} /> : null}
          {details.startedAt ? <DetailRow label="Started" value={details.startedAt} /> : null}
          {details.finishedAt ? <DetailRow label="Finished" value={details.finishedAt} /> : null}
          {details.duration ? <DetailRow label="Duration" value={details.duration} /> : null}
          {data.detail ? <DetailRow label="Ref" value={data.detail} /> : null}
        </div>

        {details.output ? (
          <div className="mt-6">
            <div className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">Status detail</div>
            <p className="mt-2 text-[13px] leading-relaxed text-foreground">{details.output}</p>
          </div>
        ) : null}

        {details.inputs?.length ? (
          <div className="mt-6">
            <div className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">Inputs</div>
            <ul className="mt-2 space-y-1.5">
              {details.inputs.map((item) => (
                <li
                  key={item}
                  className="rounded-md border border-border bg-background px-2.5 py-1.5 font-mono text-[11px] text-foreground"
                >
                  {item}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {details.logs?.length ? (
          <div className="mt-6">
            <div className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">Recent logs</div>
            <ol className="mt-2 space-y-1.5">
              {details.logs.map((line, index) => (
                <li
                  key={`${index}-${line}`}
                  className="rounded-md border border-border bg-background px-2.5 py-1.5 text-[12px] text-foreground"
                >
                  <span className="mr-2 text-muted-foreground">{index + 1}.</span>
                  {line}
                </li>
              ))}
            </ol>
          </div>
        ) : null}
      </div>
    </aside>
  );
}

export function WorkOrderCanvas({ editable = false }: { editable?: boolean } = {}) {
  const initialGraph = useMemo(() => buildWorkflowGraph(editable), [editable]);
  const [nodes, , onNodesChange] = useNodesState(initialGraph.nodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialGraph.edges);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const selectedNode = useMemo(() => nodes.find((node) => node.id === selectedId) ?? null, [nodes, selectedId]);

  const displayNodes = useMemo(
    () =>
      nodes.map((node) => ({
        ...node,
        selected: node.id === selectedId,
        draggable: editable,
      })),
    [nodes, selectedId, editable],
  );

  const onNodeClick = useCallback<NodeMouseHandler>((_event, node) => {
    setSelectedId(node.id);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedId(null);
  }, []);

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!editable) return;
      setEdges((current) =>
        addEdge(
          {
            ...connection,
            type: "smoothstep",
            style: EDGE_STYLE,
            markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#94a3b8" },
          },
          current,
        ),
      );
    },
    [editable, setEdges],
  );

  return (
    <div className="flex h-full w-full min-w-0">
      <div className="min-h-0 min-w-0 flex-1">
        <ReactFlow
          nodes={displayNodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.18 }}
          nodesDraggable={editable}
          nodesConnectable={editable}
          elementsSelectable
          onNodesChange={editable ? onNodesChange : undefined}
          onEdgesChange={editable ? onEdgesChange : undefined}
          onConnect={editable ? onConnect : undefined}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          panOnScroll
          zoomOnScroll
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{ type: "smoothstep" }}
        >
          <Background gap={20} size={1} color="#e5e5e5" />
          <Controls
            showInteractive={false}
            className="!overflow-hidden !rounded-md !border !border-border !bg-background !shadow-none"
          />
        </ReactFlow>
      </div>
      {selectedNode ? <StepDetailSidebar node={selectedNode} onClose={() => setSelectedId(null)} /> : null}
    </div>
  );
}
