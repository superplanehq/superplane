import { useCallback, useMemo, useState, type ReactNode } from "react";
import {
  Background,
  Controls,
  Handle,
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
import { Check, Circle, Loader2, X as XIcon, X } from "lucide-react";
import { useTheme } from "@/contexts/useTheme";
import githubIcon from "@/assets/icons/integrations/github.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import superplaneIcon from "@/assets/superplane.svg";
import { cn } from "@/lib/utils";

type StepStatus = "passed" | "created" | "success" | "running" | "failed" | "pending";
type ComponentProvider = "github" | "slack" | "superplane";

function normalizeStatus(status: StepStatus): "passed" | "failed" | "running" | "pending" {
  switch (status) {
    case "passed":
    case "created":
    case "success":
      return "passed";
    case "failed":
      return "failed";
    case "running":
      return "running";
    case "pending":
      return "pending";
  }
}

function statusLabel(status: StepStatus) {
  switch (normalizeStatus(status)) {
    case "passed":
      return "Passed";
    case "failed":
      return "Failed";
    case "running":
      return "Running";
    case "pending":
      return "Pending";
  }
}

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

type NodePorts = {
  target?: boolean;
  out?: boolean;
  side?: boolean;
};

type StepNodeData = {
  /** Human step label (sidebar / instance name). */
  title: string;
  /** Component type shown in the node header with the provider icon. */
  componentName: string;
  provider: ComponentProvider;
  status: StepStatus;
  kind: "step" | "action";
  detail?: string;
  meta?: string[];
  showLogs?: boolean;
  details: StepDetail;
  /** Which connection handles are in use (derived from edges). */
  ports?: NodePorts;
};

function portsForNode(nodeId: string, edges: Edge[]): NodePorts {
  let target = false;
  let out = false;
  let side = false;
  for (const edge of edges) {
    if (edge.target === nodeId) target = true;
    if (edge.source === nodeId) {
      if (edge.sourceHandle === "side") side = true;
      else out = true;
    }
  }
  return { target, out, side };
}

function statusBadgeClass(status: StepStatus) {
  switch (normalizeStatus(status)) {
    case "passed":
      return "border-[#15803d] bg-[#dcfce7] text-[#166534] dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200";
    case "running":
      return "border-[#2563eb] bg-[#dbeafe] text-[#1d4ed8] dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-200";
    case "failed":
      return "border-[#b91c1c] bg-[#fee2e2] text-[#991b1b] dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200";
    case "pending":
      return "border-[#737373] bg-[#f3f3f3] text-[#525252] dark:border-border dark:bg-muted dark:text-muted-foreground";
  }
}

function statusSentence(status: StepStatus) {
  switch (normalizeStatus(status)) {
    case "passed":
      return "Completed successfully.";
    case "failed":
      return "Failed and stopped.";
    case "running":
      return "Currently executing.";
    case "pending":
      return "Waiting to start.";
  }
}

function nodeFooterMetrics(details: StepDetail): string | null {
  const { startedAt, finishedAt, duration } = details;
  if (duration) return duration;
  if (startedAt && finishedAt) return `${startedAt} → ${finishedAt}`;
  if (startedAt) return startedAt;
  if (finishedAt) return finishedAt;
  return null;
}

function providerIconSrc(provider: ComponentProvider): string {
  switch (provider) {
    case "github":
      return githubIcon;
    case "slack":
      return slackIcon;
    case "superplane":
      return superplaneIcon;
  }
}

function nodeDescription(data: StepNodeData): string | null {
  if (data.detail) return data.detail;
  if (data.title !== data.componentName) return data.title;
  return null;
}

function StatusGlyph({ status, className }: { status: StepStatus; className?: string }) {
  const normalized = normalizeStatus(status);
  const iconClass = cn("size-3.5 shrink-0", className);

  if (normalized === "passed") {
    return (
      <span className={cn("flex size-4 shrink-0 items-center justify-center rounded-full bg-[#16a34a] text-white", className)}>
        <Check className="size-2.5" strokeWidth={3} />
      </span>
    );
  }
  if (normalized === "failed") {
    return (
      <span className={cn("flex size-4 shrink-0 items-center justify-center rounded-full bg-[#dc2626] text-white", className)}>
        <XIcon className="size-2.5" strokeWidth={3} />
      </span>
    );
  }
  if (normalized === "running") {
    return <Loader2 className={cn(iconClass, "animate-spin text-[#2563eb] dark:text-blue-300")} aria-hidden />;
  }
  return <Circle className={cn(iconClass, "text-[#a3a3a3] dark:text-muted-foreground")} strokeWidth={1.75} aria-hidden />;
}

function statusStripClass(status: StepStatus): string {
  switch (normalizeStatus(status)) {
    case "passed":
      return "border-[#dcfce7] bg-[#f0fdf4] text-[#166534] dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300";
    case "failed":
      return "border-[#fecaca] bg-[#fef2f2] text-[#991b1b] dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300";
    case "running":
      return "border-[#bfdbfe] bg-[#eff6ff] text-[#1d4ed8] dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-300";
    case "pending":
      return "border-[#e5e5e5] bg-[#fafafa] text-[#737373] dark:border-border dark:bg-muted dark:text-muted-foreground";
  }
}

/** Tinted status strip: glyph + label left, duration right. */
function NodeStatusFooter({ status, metrics }: { status: StepStatus; metrics: string | null }) {
  return (
    <div className={cn("flex items-center justify-between gap-2 border-t px-3.5 py-2", statusStripClass(status))}>
      <span className="inline-flex items-center gap-1.5 text-[11px] font-medium">
        <StatusGlyph status={status} />
        {statusLabel(status)}
      </span>
      <span className="font-mono text-[11px] tabular-nums opacity-80">{metrics ?? "—"}</span>
    </div>
  );
}

/** Card: icon + component name + tinted status footer. */
function StackNodeShell({
  data,
  selected,
  widthClass,
  children,
}: {
  data: StepNodeData;
  selected?: boolean;
  widthClass: string;
  children?: ReactNode;
}) {
  const metrics = nodeFooterMetrics(data.details);
  const description = nodeDescription(data);
  const iconSrc = providerIconSrc(data.provider);
  const monoIcon = data.provider === "github" || data.provider === "superplane";

  return (
    <div
      className={cn(
        "cursor-pointer overflow-hidden rounded-2xl border border-border bg-card shadow-[0_1px_2px_rgba(15,23,42,0.04),0_4px_12px_rgba(15,23,42,0.04)] dark:shadow-[0_1px_2px_rgba(0,0,0,0.35)]",
        widthClass,
        selected && "border-ring shadow-[0_0_0_3px_rgba(15,23,42,0.06)] dark:shadow-[0_0_0_3px_rgba(163,163,163,0.25)]",
      )}
    >
      <div className="px-3.5 pt-3.5 pb-3">
        <div className="flex items-start gap-2.5">
          <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-muted">
            <img
              src={iconSrc}
              alt=""
              className={cn("size-4", monoIcon ? "opacity-90 dark:invert" : null)}
            />
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[14px] leading-snug font-semibold tracking-[-0.015em] text-card-foreground">
              {data.componentName}
            </div>
            {description ? (
              <p className="mt-0.5 text-[12px] leading-snug text-muted-foreground">{description}</p>
            ) : null}
          </div>
        </div>
        {children}
      </div>

      <NodeStatusFooter status={data.status} metrics={metrics} />
    </div>
  );
}

const HANDLE_CLASS =
  "!size-2.5 !rounded-[3px] !border !border-border !bg-card !shadow-none";

function StepNode({ data, selected }: NodeProps<Node<StepNodeData>>) {
  const ports = data.ports;
  return (
    <div className="relative">
      {ports?.target ? <Handle type="target" position={Position.Top} className={HANDLE_CLASS} /> : null}
      <StackNodeShell data={data} selected={selected} widthClass="w-[280px]" />
      {ports?.out ? <Handle id="out" type="source" position={Position.Bottom} className={HANDLE_CLASS} /> : null}
      {ports?.side ? (
        <Handle id="side" type="source" position={Position.Right} className={cn(HANDLE_CLASS, "!top-1/2")} />
      ) : null}
    </div>
  );
}

function ActionNode({ data, selected }: NodeProps<Node<StepNodeData>>) {
  return (
    <div className="relative">
      {data.ports?.target ? <Handle type="target" position={Position.Left} className={HANDLE_CLASS} /> : null}
      <StackNodeShell data={data} selected={selected} widthClass="w-[260px]" />
    </div>
  );
}

const nodeTypes = {
  step: StepNode,
  action: ActionNode,
};

const MAIN_X = 120;
const SIDE_X = 520;
const EDGE_TYPE = "default";

function edgePalette(isDark: boolean) {
  if (isDark) {
    return {
      default: { stroke: "#4a4740", strokeWidth: 1.5 },
      running: { stroke: "#818cf8", strokeWidth: 1.5 },
      failed: { stroke: "#f87171", strokeWidth: 1.5 },
    };
  }
  return {
    default: { stroke: "#cbd5e1", strokeWidth: 1.5 },
    running: { stroke: "#818cf8", strokeWidth: 1.5 },
    failed: { stroke: "#fca5a5", strokeWidth: 1.5 },
  };
}

function backgroundColors(isDark: boolean) {
  if (isDark) {
    return { gap: 22, size: 1, color: "#33312b", bgColor: "#14120b" };
  }
  return { gap: 22, size: 1, color: "#e5e7eb", bgColor: "#f9fafb" };
}

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
        componentName: "Create Branch",
        provider: "github",
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
      { x: MAIN_X, y: 200 },
      {
        title: "Open Draft PR",
        componentName: "Create Pull Request",
        provider: "github",
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
      { x: MAIN_X, y: 400 },
      {
        title: "Write Plan",
        componentName: "Agent",
        provider: "superplane",
        status: "passed",
        detail: "Write Plan · plan.md · 12 steps",
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
      { x: SIDE_X, y: 280 },
      {
        title: "Post plan comment",
        componentName: "Create Issue Comment",
        provider: "github",
        status: "failed",
        detail: "comment · #8841",
        details: {
          startedAt: "14:03:42",
          finishedAt: "14:03:43",
          duration: "1s",
          agent: "github-worker",
          attempt: "1 / 1",
          output: "GitHub API returned 403: Resource not accessible by integration.",
          logs: ["Rendered markdown comment", "Posted via Issues API", "Received 403 Forbidden"],
          inputs: ["issue: #8841"],
        },
      },
    ),
    action(
      "action-notify-start",
      { x: SIDE_X, y: 440 },
      {
        title: "Notify work started",
        componentName: "Send Text Message",
        provider: "slack",
        status: "passed",
        detail: "#factory · work started",
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
      { x: SIDE_X, y: 600 },
      {
        title: "Add in-progress label",
        componentName: "Add Issue Label",
        provider: "github",
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
      { x: SIDE_X, y: 760 },
      {
        title: "Correlate PR Memory",
        componentName: "Add Memory",
        provider: "superplane",
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
      { x: MAIN_X, y: 900 },
      {
        title: "Implementation",
        componentName: "Agent",
        provider: "superplane",
        status: "running",
        detail: "Implementation · commits · 4 files",
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
      { x: SIDE_X, y: 920 },
      {
        title: "Mark Plan Done",
        componentName: "Update Work Order Status",
        provider: "superplane",
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
      { x: MAIN_X, y: 1200 },
      {
        title: "Babysit",
        componentName: "Babysit",
        provider: "superplane",
        status: "pending",
        detail: "Software Factory · Timeout: 1h",
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
      { x: SIDE_X, y: 1180 },
      {
        title: "Mark Implementation Done",
        componentName: "Update Work Order Status",
        provider: "superplane",
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
      { x: SIDE_X, y: 1340 },
      {
        title: "Notify work failed",
        componentName: "Send Text Message",
        provider: "slack",
        status: "pending",
        detail: "#factory · on babysit failed",
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
      { x: SIDE_X, y: 1500 },
      {
        title: "Mark Implementation Failed Memory",
        componentName: "Add Memory",
        provider: "superplane",
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
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-pr-plan",
      source: "open-draft-pr",
      sourceHandle: "out",
      target: "write-plan",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-plan-comment",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-comment",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-plan-notify",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-notify-start",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-plan-label",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-label",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-plan-correlate",
      source: "write-plan",
      sourceHandle: "side",
      target: "action-correlate",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-plan-impl",
      source: "write-plan",
      sourceHandle: "out",
      target: "implementation",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-impl-mark-plan",
      source: "implementation",
      sourceHandle: "side",
      target: "action-mark-plan",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-impl-babysit",
      source: "implementation",
      sourceHandle: "out",
      target: "babysit",
      type: EDGE_TYPE,
      animated: true,
      data: { tone: "running" },
    },
    {
      id: "e-babysit-mark",
      source: "babysit",
      sourceHandle: "side",
      target: "action-mark-impl",
      type: EDGE_TYPE,
      data: { tone: "default" },
    },
    {
      id: "e-babysit-notify-fail",
      source: "babysit",
      sourceHandle: "side",
      target: "action-notify-fail",
      type: EDGE_TYPE,
      data: { tone: "failed" },
    },
    {
      id: "e-babysit-fail-memory",
      source: "babysit",
      sourceHandle: "side",
      target: "action-mark-fail-memory",
      type: EDGE_TYPE,
      data: { tone: "failed" },
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
    <aside className="flex h-full w-[320px] shrink-0 flex-col border-l border-border bg-muted">
      <div className="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <img
              src={providerIconSrc(data.provider)}
              alt=""
              className="size-3.5 shrink-0"
            />
            <div className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">
              {data.componentName}
            </div>
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
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const palette = useMemo(() => edgePalette(isDark), [isDark]);
  const background = useMemo(() => backgroundColors(isDark), [isDark]);

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
        data: {
          ...node.data,
          ports: portsForNode(node.id, edges),
        },
      })),
    [nodes, edges, selectedId, editable],
  );

  const displayEdges = useMemo(
    () =>
      edges.map((edge) => {
        const tone = (edge.data as { tone?: "default" | "running" | "failed" } | undefined)?.tone ?? "default";
        return {
          ...edge,
          style: palette[tone],
        };
      }),
    [edges, palette],
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
            type: EDGE_TYPE,
            data: { tone: "default" },
            style: palette.default,
          },
          current,
        ),
      );
    },
    [editable, setEdges, palette.default],
  );

  return (
    <div className="flex h-full w-full min-w-0">
      <div className="min-h-0 min-w-0 flex-1">
        <ReactFlow
          nodes={displayNodes}
          edges={displayEdges}
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
          defaultEdgeOptions={{ type: EDGE_TYPE, style: palette.default }}
          colorMode={isDark ? "dark" : "light"}
        >
          <Background gap={background.gap} size={background.size} color={background.color} bgColor={background.bgColor} />
          <Controls
            showInteractive={false}
            className="!overflow-hidden !rounded-xl !border !border-border !bg-card !shadow-[0_1px_3px_rgba(15,23,42,0.06)]"
          />
        </ReactFlow>
      </div>
      {selectedNode ? <StepDetailSidebar node={selectedNode} onClose={() => setSelectedId(null)} /> : null}
    </div>
  );
}
