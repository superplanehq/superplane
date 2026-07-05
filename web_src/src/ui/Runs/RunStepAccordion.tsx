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
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import { RunStepConfigView } from "./RunStepConfigView";
import { RunStepTimeline } from "./RunStepTimeline";
import { buildExecutionChain, eventBadgeForExecution, eventBadgeForTriggeredTrigger } from "./runNodeDetailModel";
import { formatEventTimestamp, formatStepDuration, getStepActivity } from "./runSummary";

/** Controls what expanded steps render: the run-detail timeline or the read-only step configuration. */
export type StepDetailMode = "run-details" | "step-config";

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
  stepDetailMode = "run-details",
}: {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
  stepDetailMode?: StepDetailMode;
}) {
  if (stepDetailMode === "step-config") {
    return <RunStepConfigView nodeId={nodeId} workflowNodes={workflowNodes} componentIconMap={componentIconMap} />;
  }

  return (
    <RunStepTimeline
      run={run}
      nodeId={nodeId}
      workflowNodes={workflowNodes}
      componentIconMap={componentIconMap}
      executions={executions}
    />
  );
}

/**
 * Per-step action affordance shown only on in-flight (running / waiting) steps: a
 * single "Action" button whose hover card lists the contextual actions — an
 * approval step offers Approve / Reject / Cancel, any other in-flight step offers Stop.
 */
function StepActionMenu({ isApproval }: { isApproval: boolean }) {
  const actions: { label: string; tone: "approve" | "danger" | "default" }[] = isApproval
    ? [
        { label: "Approve", tone: "approve" },
        { label: "Reject", tone: "danger" },
        { label: "Cancel", tone: "default" },
      ]
    : [{ label: "Stop", tone: "danger" }];

  return (
    <HoverCard openDelay={80} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          onClick={(event) => event.stopPropagation()}
          className="shrink-0 rounded border border-slate-200 bg-white px-1.5 py-0.5 text-[11px] font-medium text-slate-600 transition-colors hover:bg-slate-50 hover:text-slate-800 data-[state=open]:bg-slate-100 data-[state=open]:text-slate-800"
        >
          Action
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="end" side="bottom" sideOffset={4} className="w-40 p-1">
        <div className="flex flex-col">
          {actions.map((action) => (
            <button
              key={action.label}
              type="button"
              onClick={(event) => event.stopPropagation()}
              className={cn(
                "rounded px-2 py-1.5 text-left text-[13px] transition-colors",
                action.tone === "approve" && "text-emerald-700 hover:bg-emerald-50",
                action.tone === "danger" && "text-red-600 hover:bg-red-50",
                action.tone === "default" && "text-slate-600 hover:bg-slate-50",
              )}
            >
              {action.label}
            </button>
          ))}
        </div>
      </HoverCardContent>
    </HoverCard>
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

  // Only in-flight steps (running / waiting) expose an inline action; the run as a
  // whole is stopped/rerun from the run header, not per step.
  const activity = !isTrigger && execution ? getStepActivity(workflowNode, execution) : "done";
  const showStepAction = activity === "running" || activity === "waiting";
  const isApproval = workflowNode?.component === "approval";

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
      {showStepAction ? <StepActionMenu isApproval={isApproval} /> : null}
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
