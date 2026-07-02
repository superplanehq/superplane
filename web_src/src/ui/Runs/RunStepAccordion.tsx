import JsonView from "@uiw/react-json-view";
import { AlertTriangle, Check, ChevronRight, Copy, Maximize2, Sparkles } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import {
  buildExecutionChain,
  eventBadgeForExecution,
  eventBadgeForTriggeredTrigger,
  hasObjectValue,
  isErrorValue,
} from "./runNodeDetailModel";
import { useRunNodeDetailPresentation } from "./useRunNodeDetailPresentation";

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

function HeaderIconButton({
  label,
  icon,
  onClick,
}: {
  label: string;
  icon: ReactNode;
  onClick?: (event: React.MouseEvent) => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          onClick={(event) => {
            event.stopPropagation();
            onClick?.(event);
          }}
          className="flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-700"
        >
          {icon}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

export function JsonDetailBox({ title, value }: { title: string; value: unknown }) {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    void navigator.clipboard?.writeText(JSON.stringify(value, null, 2)).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const actions = (
    <>
      <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
      <HeaderIconButton
        label={copied ? "Copied" : "Copy"}
        icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
        onClick={handleCopy}
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
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <div className="max-h-[70vh] overflow-auto">
            <JsonView value={value as object} collapsed={false} displayDataTypes={false} style={{ fontSize: 12 }} />
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

function ErrorDetailBox({
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
      <div className="flex items-center gap-1.5 border-b border-red-200 bg-red-50 px-3 py-1.5">
        <AlertTriangle className="h-3.5 w-3.5 text-red-600" />
        <span className="text-[11px] font-semibold uppercase tracking-wide text-red-600">Error</span>
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

function buildSummaryDetails(rawDetails: Record<string, unknown>, metadata: Record<string, unknown> | undefined) {
  if (!isErrorValue(rawDetails.Error)) return rawDetails;
  return Object.fromEntries(
    Object.entries(rawDetails).filter(([key]) => key !== "Error" && !(key in (metadata ?? {}))),
  );
}

function NodeOutcomeBox({
  presentation,
  execution,
}: {
  presentation: ReturnType<typeof useRunNodeDetailPresentation>;
  execution: CanvasesCanvasNodeExecution | undefined;
}) {
  const errorValue = presentation.tabData?.details?.Error;
  if (isErrorValue(errorValue)) {
    return (
      <ErrorDetailBox message={errorValue.message} reason={execution?.resultReason} metadata={execution?.metadata} />
    );
  }
  if (presentation.hasPayload) {
    return <JsonDetailBox title="Output" value={presentation.tabData?.payload} />;
  }
  return null;
}

export function AccordionNodeDetail({
  run,
  nodeId,
  workflowNodes,
  executions,
}: {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes: ComponentsNode[];
  executions: CanvasesCanvasNodeExecution[];
}) {
  const presentation = useRunNodeDetailPresentation({ run, nodeId, workflowNodes, executions });

  const execution = executions.find((item) => item.nodeId === nodeId);
  const inputData = presentation.isTriggerNode ? undefined : execution?.rootEvent?.data;
  const hasInput = hasObjectValue(inputData);
  const summaryDetails = buildSummaryDetails(presentation.tabData?.details ?? {}, execution?.metadata);

  if (!presentation.hasAnyTab && !hasInput) {
    return <div className="px-3 py-3 text-xs text-gray-400">No execution data for this node in this run.</div>;
  }

  return (
    <div className="flex flex-col gap-2 bg-slate-50 px-3 py-3">
      {presentation.hasDetailsSection ? (
        <DetailBox
          title="Summary"
          actions={<HeaderIconButton label="Ask agent" icon={<Sparkles className="h-3.5 w-3.5" />} />}
        >
          <RunNodeDetailDetailsView
            details={summaryDetails}
            statusBadge={presentation.headerEventBadge}
            relativeTime={presentation.createdAt}
          />
        </DetailBox>
      ) : null}
      {hasInput ? <JsonDetailBox title="Input" value={inputData} /> : null}
      {presentation.hasConfig ? (
        <JsonDetailBox title="Runtime Config" value={presentation.tabData?.configuration} />
      ) : null}
      <NodeOutcomeBox presentation={presentation} execution={execution} />
    </div>
  );
}

export function AccordionRow({
  nodeId,
  workflowNode,
  componentIconMap,
  execution,
  isTrigger,
  isExpanded,
  onToggle,
  className,
}: {
  nodeId: string;
  workflowNode?: ComponentsNode;
  componentIconMap: Record<string, string>;
  execution?: CanvasesCanvasNodeExecution;
  isTrigger: boolean;
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
              isExpanded={expandedNodeId === nodeId}
              onToggle={onToggleNode}
              className={rowClassName}
            />
            {expandedNodeId === nodeId ? (
              <AccordionNodeDetail run={run} nodeId={nodeId} workflowNodes={workflowNodes} executions={executions} />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
