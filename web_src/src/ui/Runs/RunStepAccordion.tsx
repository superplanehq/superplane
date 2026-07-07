import JsonView from "@uiw/react-json-view";
import { AlertTriangle, Check, ChevronRight, Copy, Maximize2, Sparkles } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import Editor from "@monaco-editor/react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import {
  approvalEvents,
  cursorAgentEvents,
  githubEvents,
  memoryEvents,
  runBashEvents,
} from "./storybooks/timelineGroupsFixtures";
import { EventTimeline, type RuntimeConfigNode, type TimelineEvent } from "./storybooks/timelineGroupsModel";
import { buildExecutionChain, eventBadgeForExecution, eventBadgeForTriggeredTrigger } from "./runNodeDetailModel";
import { formatEventTimestamp, formatStepDuration, getStepActivity } from "./runSummary";

export function StatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

export function DetailBox({ title, children, actions }: { title: string; children: ReactNode; actions?: ReactNode }) {
  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white">
      <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{title}</span>
        {actions ? <div className="flex items-center gap-0.5">{actions}</div> : null}
      </div>
      <div className="px-3 py-2.5">{children}</div>
    </div>
  );
}

export function HeaderIconButton({
  label,
  icon,
  onClick,
  active,
}: {
  label: string;
  icon: ReactNode;
  onClick?: (event: React.MouseEvent) => void;
  /** When defined, renders as a toggle: reflects pressed state and shows a filled background when on. */
  active?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-pressed={active}
          onClick={(event) => {
            event.stopPropagation();
            onClick?.(event);
          }}
          className={cn(
            "flex h-6 w-6 items-center justify-center rounded transition-colors",
            active
              ? "bg-blue-100 text-blue-700 hover:bg-blue-200"
              : "text-slate-400 hover:bg-slate-200 hover:text-slate-700",
          )}
        >
          {icon}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

/** Read-only JSON viewer with line folding, sized to fill its container. */
export function PayloadMonaco({ value }: { value: string }) {
  return (
    <Editor
      height="100%"
      defaultLanguage="json"
      value={value}
      theme="vs"
      options={{
        readOnly: true,
        domReadOnly: true,
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: "on",
        wordWrap: "on",
        folding: true,
        showFoldingControls: "always",
        scrollBeyondLastLine: false,
        renderWhitespace: "none",
        scrollbar: { vertical: "auto", horizontal: "auto" },
      }}
    />
  );
}

export function JsonDetailBox({
  title,
  value,
  nodeName,
  nodeIcon,
}: {
  title: string;
  value: unknown;
  /** Step/node name shown in the expanded modal header to indicate the source. */
  nodeName?: string;
  /** Step/node icon shown in the expanded modal header. */
  nodeIcon?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [modalCopied, setModalCopied] = useState(false);
  const payloadString = useMemo(() => JSON.stringify(value, null, 2), [value]);

  const copyPayload = (markCopied: (copied: boolean) => void) => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    markCopied(true);
    setTimeout(() => markCopied(false), 1500);
  };

  const actions = (
    <>
      <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
      <HeaderIconButton
        label={copied ? "Copied" : "Copy"}
        icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
        onClick={() => copyPayload(setCopied)}
      />
      <HeaderIconButton label="Expand" icon={<Maximize2 className="h-3.5 w-3.5" />} onClick={() => setOpen(true)} />
    </>
  );

  return (
    <>
      <DetailBox title={title} actions={actions}>
        <JsonView value={value as object} collapsed={2} displayDataTypes={false} style={{ fontSize: 12 }} />
      </DetailBox>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="large"
          className="flex h-[80vh] w-[60vw] max-w-[60vw] flex-col gap-0 overflow-hidden p-0"
          onClick={(event) => event.stopPropagation()}
        >
          <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10">
            <div className="flex min-w-0 items-center gap-1.5">
              {nodeIcon}
              {nodeName ? <span className="truncate text-[12px] font-medium text-slate-700">{nodeName}</span> : null}
              <DialogTitle className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                {title}
              </DialogTitle>
            </div>
            <div className="flex items-center gap-0.5">
              <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
              <HeaderIconButton
                label={modalCopied ? "Copied" : "Copy"}
                icon={
                  modalCopied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />
                }
                onClick={() => copyPayload(setModalCopied)}
              />
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-hidden">
            <PayloadMonaco value={payloadString} />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

function humanizeReason(reason?: string) {
  if (!reason) return undefined;
  const cleaned = reason
    .replace(/^RESULT_REASON_/, "")
    .replace(/_/g, " ")
    .toLowerCase();
  return cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
}

export function ErrorDetailBox({
  message,
  reason,
  metadata,
}: {
  message?: string;
  reason?: string;
  metadata?: Record<string, unknown>;
}) {
  const humanizedReason = humanizeReason(reason);
  const metadataEntries = metadata ? Object.entries(metadata) : [];

  return (
    <div className="overflow-hidden rounded border border-red-200 bg-white">
      <div className="flex items-center justify-between gap-1.5 border-b border-red-200 bg-red-50 px-3 py-1.5">
        <span className="flex min-w-0 items-center gap-1.5">
          <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-red-600" />
          <span className="truncate text-[11px] font-semibold uppercase tracking-wide text-red-600">
            Error - Output not emitted
          </span>
        </span>
        <HeaderIconButton label="Ask agent" icon={<Sparkles className="h-3.5 w-3.5" />} />
      </div>
      <div className="flex flex-col gap-1.5 px-3 py-2.5 text-[13px]">
        {message ? <span className="min-w-0 break-all font-medium text-red-600">{message}</span> : null}
        {humanizedReason ? (
          <div className="flex items-start gap-2">
            <span className="shrink-0 text-left text-gray-500">Reason:</span>
            <span className="min-w-0 break-all text-gray-800">{humanizedReason}</span>
          </div>
        ) : null}
        {metadataEntries.map(([key, value]) => (
          <div key={key} className="flex items-start gap-2">
            <span className="shrink-0 text-left text-gray-500" title={key}>
              {key}:
            </span>
            <span className="min-w-0 break-all text-gray-800">
              {typeof value === "object" ? JSON.stringify(value) : String(value ?? "")}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function AccordionNodeDetail({
  run,
  nodeId,
  workflowNodes,
  componentIconMap = {},
  executions,
}: {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
}) {
  // WIREFRAME PREVIEW (never merged to production): render the flat timeline-events
  // design with mocked data instead of the real RunStepTimeline, so it can be reviewed
  // inside the live-canvas run inspection panel. The scenario shape is picked by
  // component type, but the *outcome* follows the node's real execution result: steps
  // that errored render an error terminal instead of the mocked Output/Summary cards.
  // `run` stays in the props contract but is unused by the mocked wireframe.
  void run;
  const node = workflowNodes.find((item) => item.id === nodeId);
  const component = (node?.component ?? "").toLowerCase();
  const execution = executions.find((item) => item.nodeId === nodeId);
  const base = pickWireframeScenario(component);
  const wireframeEvents = executionErrored(execution) ? applyErrorTerminal(base, execution) : base;
  // Feed the real node into the Runtime Config card so its read-only form has schema/values.
  const configNode: RuntimeConfigNode | undefined = node
    ? {
        component: node.component,
        name: node.name,
        configuration: node.configuration as Record<string, unknown> | undefined,
        iconSrc: getHeaderIconSrc(node.component),
        iconSlug: node.component ? componentIconMap[node.component] : undefined,
      }
    : undefined;
  return <EventTimeline events={wireframeEvents} configNode={configNode} />;
}

/** Whether a step's real execution resolved as an error (matching the app's own definition). */
function executionErrored(execution?: CanvasesCanvasNodeExecution): boolean {
  if (!execution) return false;
  return (
    execution.resultReason === "RESULT_REASON_ERROR" ||
    (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED")
  );
}

/**
 * Rewrites a scenario's tail so an errored step reads as a failure: the mocked terminal
 * Output/Summary cards become an error card ("Error - Output not emitted", carrying the
 * real message/reason), and the last running activity stops showing a success. A network
 * line flips to a red error status/response; a plain success line (e.g. "exit 0",
 * "Stored key …") is dropped so the error card speaks instead. Input/queue/config stay.
 */
function applyErrorTerminal(events: TimelineEvent[], execution?: CanvasesCanvasNodeExecution): TimelineEvent[] {
  const message = execution?.resultMessage || "The step errored and did not emit an output.";
  const withoutTerminal = events.filter(
    (event) => !(event.type === "card" && (event.id === "output" || event.id === "summary" || event.id === "error")),
  );

  const lastActivity = lastActivityIndex(withoutTerminal);
  const rewritten = withoutTerminal.flatMap((event, index) => {
    if (index !== lastActivity) return [event];
    if (event.type === "line" && event.line.request) {
      return [
        {
          ...event,
          line: {
            ...event.line,
            dotClassName: "bg-red-500",
            request: {
              ...event.line.request,
              status: 500,
              statusText: "Internal Server Error",
              responseBody: { error: message },
            },
          },
        } satisfies TimelineEvent,
      ];
    }
    // A non-network success line (exit 0 / stored / etc.) would misrepresent the error, so drop it.
    return [];
  });

  return [
    ...rewritten,
    {
      type: "card",
      id: "error",
      card: {
        kind: "error",
        message,
        reason: execution?.resultReason || "RESULT_REASON_ERROR",
      },
    },
  ];
}

/**
 * Index of the last "running result" activity in a scenario — the last network/line event
 * or logs card, ignoring lifecycle markers (queue enter/exit, waiting) and payload cards.
 */
function lastActivityIndex(events: TimelineEvent[]): number {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.type === "line" && !["q-enter", "q-exit", "waiting"].includes(event.id)) return index;
    if (event.type === "card" && event.card.kind === "logs") return index;
  }
  return -1;
}

/** Maps a node's component to the mocked wireframe scenario shape (successful path). */
function pickWireframeScenario(component: string): TimelineEvent[] {
  if (component.includes("bash") || component.includes("runner")) return runBashEvents;
  if (component.includes("agent") || component.includes("cursor")) return cursorAgentEvents;
  if (component.includes("memory")) return memoryEvents;
  if (component.includes("github")) return githubEvents;
  return approvalEvents;
}

type StepActionTone = "approve" | "danger" | "neutral";
type StepAction = { label: string; tone: StepActionTone };

/**
 * The inline action buttons a step row exposes, driven by whether it's the trigger and
 * the step's activity/component:
 * - trigger and terminal steps (done / error) → "Rerun"
 * - running / waiting steps → "Stop"
 * - an in-flight approval additionally offers "Approve" / "Reject"
 */
function getStepActions(
  isTrigger: boolean,
  activity: ReturnType<typeof getStepActivity>,
  isApproval: boolean,
): StepAction[] {
  if (isTrigger) return [{ label: "Rerun", tone: "neutral" }];

  if (activity === "running" || activity === "waiting") {
    if (isApproval) {
      return [
        { label: "Approve", tone: "approve" },
        { label: "Reject", tone: "danger" },
        { label: "Stop", tone: "danger" },
      ];
    }
    return [{ label: "Stop", tone: "danger" }];
  }

  return [{ label: "Rerun", tone: "neutral" }];
}

function StepActionButton({ label, tone }: StepAction) {
  return (
    <button
      type="button"
      onClick={(event) => event.stopPropagation()}
      className={cn(
        "shrink-0 rounded border px-1.5 py-0.5 text-[11px] font-medium transition-colors",
        tone === "approve" && "border-emerald-200 bg-white text-emerald-700 hover:bg-emerald-50",
        tone === "danger" && "border-red-200 bg-white text-red-600 hover:bg-red-50",
        tone === "neutral" && "border-slate-200 bg-white text-slate-600 hover:bg-slate-50 hover:text-slate-800",
      )}
    >
      {label}
    </button>
  );
}

export function AccordionRow({
  nodeId,
  workflowNode,
  componentIconMap,
  execution,
  isTrigger,
  triggerTimestamp,
  isExpanded,
  onToggle,
  className,
}: {
  nodeId: string;
  workflowNode?: ComponentsNode;
  componentIconMap: Record<string, string>;
  execution?: CanvasesCanvasNodeExecution;
  isTrigger: boolean;
  /** When the triggering event was received (run start); shown in place of a duration for the trigger row. */
  triggerTimestamp?: string;
  isExpanded: boolean;
  onToggle: (nodeId: string) => void;
  className?: string;
}) {
  const iconSrc = getHeaderIconSrc(workflowNode?.component);
  const iconSlug = workflowNode?.component ? componentIconMap[workflowNode.component] : undefined;
  const nodeName = workflowNode?.name || nodeId;

  const badge = isTrigger
    ? eventBadgeForTriggeredTrigger(workflowNode)
    : execution
      ? eventBadgeForExecution(workflowNode, execution)
      : null;

  // Triggers have no duration; instead show when their event was received.
  const meta = isTrigger ? formatEventTimestamp(triggerTimestamp) : execution ? formatStepDuration(execution) : null;

  const activity = !isTrigger && execution ? getStepActivity(workflowNode, execution) : "done";
  const isApproval = workflowNode?.component === "approval";
  const stepActions = getStepActions(isTrigger, activity, isApproval);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onToggle(nodeId)}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        onToggle(nodeId);
      }}
      className={cn(
        "flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left transition-colors",
        isExpanded ? "bg-sky-100" : "hover:bg-gray-50",
        className,
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
        alt={nodeName}
        size={RUN_NODE_ICON_SIZE}
        className={cn("h-3.5 w-3.5 shrink-0", isExpanded ? "text-gray-800" : "text-gray-500")}
      />
      <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-gray-800">{nodeName}</span>
      {stepActions.map((action) => (
        <StepActionButton key={action.label} label={action.label} tone={action.tone} />
      ))}
      <HeaderIconButton label="Send to agent" icon={<Sparkles className="h-3.5 w-3.5" />} />
      {meta ? <span className="shrink-0 text-[11px] tabular-nums text-gray-600">{meta}</span> : null}
      {badge ? <StatusBadge badgeColor={badge.badgeColor} label={badge.label} /> : null}
    </div>
  );
}

export function AccordionNodeList({
  canvasId,
  run,
  workflowNodes,
  componentIconMap = {},
  expandedNodeId,
  onToggleNode,
  rowClassName,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  rowClassName?: string;
}) {
  const executionsQuery = useEventExecutions(canvasId, run.rootEvent?.id || null);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const triggerNodeId = run.rootEvent?.nodeId;
  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);

  if (executionsQuery.isLoading) {
    return <p className="px-3 py-4 text-xs text-gray-400">Loading nodes...</p>;
  }

  if (executionChain.length === 0) {
    return <p className="px-3 py-4 text-xs text-gray-400">No executed nodes in this run.</p>;
  }

  return (
    <div className="divide-y divide-slate-950/10">
      {executionChain.map((nodeId) => {
        const isTrigger = nodeId === triggerNodeId;
        const workflowNode = workflowNodes.find((node) => node.id === nodeId);
        const execution = executions.find((item) => item.nodeId === nodeId);

        return (
          <div key={nodeId}>
            <AccordionRow
              nodeId={nodeId}
              workflowNode={workflowNode}
              componentIconMap={componentIconMap}
              execution={execution}
              isTrigger={isTrigger}
              triggerTimestamp={run.rootEvent?.createdAt ?? run.createdAt}
              isExpanded={expandedNodeId === nodeId}
              onToggle={onToggleNode}
              className={rowClassName}
            />
            {expandedNodeId === nodeId ? (
              <AccordionNodeDetail
                run={run}
                nodeId={nodeId}
                workflowNodes={workflowNodes}
                componentIconMap={componentIconMap}
                executions={executions}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
