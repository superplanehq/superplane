/* eslint-disable max-lines-per-function, complexity */
import JsonView from "@uiw/react-json-view";
import { ChevronLeft, ChevronRight, Loader2, X } from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn, flattenObject, resolveIcon } from "@/lib/utils";
import { getExecutionDetails } from "@/pages/workflowv2/mappers";
import { getExecutionStatus } from "@/ui/Runs/runPresentation";

interface RunNodeDetailModalProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes?: ComponentsNode[];
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
}

type TabKey = "details" | "payload" | "configuration";

type TabData = {
  details?: Record<string, unknown>;
  payload?: unknown;
  configuration?: unknown;
};

function StatusPill({ className, label }: { className: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded px-1.5 py-[1px] text-[10px] font-semibold uppercase tracking-wide ring-0",
        className,
      )}
    >
      {label}
    </span>
  );
}

function buildExecutionTabData(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: ComponentsNode | undefined,
  workflowNodes: ComponentsNode[],
): TabData {
  const tabData: TabData = {};
  let details: Record<string, unknown> = {};
  const componentName = typeof workflowNode?.component === "string" ? workflowNode.component : undefined;

  if (componentName && workflowNode) {
    const customDetails = getExecutionDetails(componentName, execution, workflowNode, workflowNodes);
    if (customDetails && Object.keys(customDetails).length > 0) {
      details = { ...customDetails };
    }
  }

  if (Object.keys(details).length === 0) {
    const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
    details = { ...flattenObject((hasOutputs ? execution.outputs : execution.metadata) || {}) };
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED") &&
    !("Error" in details)
  ) {
    details.Error = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  if (execution.result === "RESULT_CANCELLED" && !("Cancelled by" in details)) {
    const cancelledBy = execution.cancelledBy;
    details["Cancelled by"] = cancelledBy?.name || cancelledBy?.id || "Unknown";
  }

  tabData.details = Object.fromEntries(
    Object.entries(details).filter(([, value]) => value !== undefined && value !== "" && value !== null),
  );

  tabData.payload = extractExecutionPayload(execution);

  if (execution.configuration && Object.keys(execution.configuration).length > 0) {
    tabData.configuration = execution.configuration;
  }

  return tabData;
}

function extractExecutionPayload(execution: CanvasesCanvasNodeExecution): unknown {
  if (!execution.outputs || Object.keys(execution.outputs).length === 0) {
    return undefined;
  }

  const outputData = Object.values(execution.outputs).find((output) => Array.isArray(output) && output.length > 0) as
    | unknown[]
    | undefined;
  if (outputData && outputData.length > 0) {
    return outputData[0];
  }

  return execution.outputs;
}

function buildTriggerTabData(run: CanvasesCanvasRun, workflowNode: ComponentsNode | undefined): TabData {
  const details: Record<string, unknown> = {};
  const rootEvent = run.rootEvent;

  if (rootEvent?.channel) details.Channel = rootEvent.channel;
  if (rootEvent?.customName) details.Name = rootEvent.customName;
  if (rootEvent?.createdAt) details["Triggered at"] = rootEvent.createdAt;

  const tabData: TabData = {
    details: Object.keys(details).length > 0 ? details : undefined,
    payload: rootEvent?.data && Object.keys(rootEvent.data).length > 0 ? rootEvent.data : undefined,
  };

  if (
    workflowNode?.configuration &&
    typeof workflowNode.configuration === "object" &&
    Object.keys(workflowNode.configuration).length > 0
  ) {
    tabData.configuration = workflowNode.configuration;
  }

  return tabData;
}

function isErrorValue(value: unknown): value is { __type: "error"; message: string } {
  return !!value && typeof value === "object" && (value as { __type?: string }).__type === "error";
}

function hasObjectValue(value: unknown) {
  return !!value && typeof value === "object" && Object.keys(value).length > 0;
}

export function RunNodeDetailModal({
  canvasId,
  run,
  nodeId,
  workflowNodes = [],
  onClose,
  onNavigateNode,
}: RunNodeDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabKey>("details");
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const triggerNodeId = run.rootEvent?.nodeId;
  const isTriggerNode = nodeId === triggerNodeId;
  const nodeExecution = useMemo(
    () => executions.find((execution) => execution.nodeId === nodeId),
    [executions, nodeId],
  );
  const workflowNode = useMemo(() => workflowNodes.find((node) => node.id === nodeId), [workflowNodes, nodeId]);

  const executionChain = useMemo(() => {
    const chain: string[] = [];
    const visited = new Set<string>();

    if (triggerNodeId) {
      chain.push(triggerNodeId);
      visited.add(triggerNodeId);
    }

    for (const execution of executions) {
      if (execution.nodeId && !visited.has(execution.nodeId)) {
        visited.add(execution.nodeId);
        chain.push(execution.nodeId);
      }
    }

    return chain;
  }, [executions, triggerNodeId]);

  const currentIndex = executionChain.indexOf(nodeId);
  const previousNodeId = currentIndex > 0 ? executionChain[currentIndex - 1] : null;
  const nextNodeId =
    currentIndex >= 0 && currentIndex < executionChain.length - 1 ? executionChain[currentIndex + 1] : null;
  const nodeName = workflowNode?.name || nodeId;
  const createdAt = isTriggerNode ? run.rootEvent?.createdAt : nodeExecution?.createdAt;

  const tabData = useMemo<TabData | null>(() => {
    if (isTriggerNode) {
      return buildTriggerTabData(run, workflowNode);
    }

    if (!nodeExecution) return null;
    return buildExecutionTabData(nodeExecution, workflowNode, workflowNodes);
  }, [isTriggerNode, nodeExecution, run, workflowNode, workflowNodes]);

  useEffect(() => {
    setActiveTab("details");
  }, [nodeId]);

  useEffect(() => {
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onClose]);

  const hasDetails = !!tabData?.details && Object.keys(tabData.details).length > 0;
  const hasPayload = hasObjectValue(tabData?.payload);
  const hasConfig = hasObjectValue(tabData?.configuration);
  const hasAnyTab = hasDetails || hasPayload || hasConfig;

  const status = nodeExecution ? getExecutionStatus(nodeExecution) : null;

  useEffect(() => {
    if (activeTab === "details" && hasDetails) return;
    if (activeTab === "payload" && hasPayload) return;
    if (activeTab === "configuration" && hasConfig) return;

    if (hasDetails) {
      setActiveTab("details");
      return;
    }

    if (hasPayload) {
      setActiveTab("payload");
      return;
    }

    if (hasConfig) {
      setActiveTab("configuration");
    }
  }, [activeTab, hasConfig, hasDetails, hasPayload]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="flex max-h-[80vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow-2xl"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={`${nodeName} run details`}
        data-testid="run-node-detail-modal"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-1.5">
          <div className="flex min-w-0 items-center gap-3">
            {onNavigateNode ? (
              <div className="flex items-center gap-0.5">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0"
                  onClick={() => previousNodeId && onNavigateNode(previousNodeId)}
                  disabled={!previousNodeId}
                >
                  <ChevronLeft className="h-3.5 w-3.5" />
                </Button>
                <span className="text-[11px] tabular-nums text-gray-400">
                  {currentIndex >= 0 ? `${currentIndex + 1}/${executionChain.length}` : ""}
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0"
                  onClick={() => nextNodeId && onNavigateNode(nextNodeId)}
                  disabled={!nextNodeId}
                >
                  <ChevronRight className="h-3.5 w-3.5" />
                </Button>
              </div>
            ) : null}
            <h3 className="truncate text-sm font-medium text-gray-900">{nodeName}</h3>
            {isTriggerNode ? (
              <StatusPill className="bg-purple-500 text-white" label="Triggered" />
            ) : status ? (
              <StatusPill className={status.className} label={status.label} />
            ) : null}
            {createdAt ? (
              <span className="text-xs text-gray-400">
                <TimeAgo date={createdAt} />
              </span>
            ) : null}
          </div>
          <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        {executionsQuery.isLoading && !isTriggerNode ? (
          <div className="flex items-center justify-center gap-2 px-4 py-8 text-xs text-gray-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading run details...
          </div>
        ) : hasAnyTab ? (
          <>
            <div className="flex h-8 shrink-0 items-center border-b border-slate-200 px-2">
              {hasDetails ? (
                <TabButton
                  active={activeTab === "details"}
                  icon="info"
                  label="Details"
                  onClick={() => setActiveTab("details")}
                />
              ) : null}
              {hasPayload ? (
                <TabButton
                  active={activeTab === "payload"}
                  icon="code"
                  label="Payload"
                  onClick={() => setActiveTab("payload")}
                />
              ) : null}
              {hasConfig ? (
                <TabButton
                  active={activeTab === "configuration"}
                  icon="settings"
                  label="Config"
                  onClick={() => setActiveTab("configuration")}
                />
              ) : null}
            </div>

            <div className="flex-1 overflow-y-auto px-4 py-3">
              {activeTab === "details" && tabData?.details ? <DetailsView details={tabData.details} /> : null}
              {activeTab === "payload" && hasPayload ? (
                <JsonView value={tabData?.payload as object} collapsed={2} style={{ fontSize: 12 }} />
              ) : null}
              {activeTab === "configuration" && hasConfig ? (
                <JsonView value={tabData?.configuration as object} collapsed={2} style={{ fontSize: 12 }} />
              ) : null}
            </div>
          </>
        ) : (
          <div className="px-4 py-6 text-center text-xs text-gray-400">
            No execution data for this node in this run.
          </div>
        )}
      </div>
    </div>
  );
}

function TabButton({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center gap-1 border-b px-2.5 py-1.5 text-[13px] font-medium",
        active ? "border-gray-800 text-gray-800" : "border-transparent text-gray-500 hover:text-gray-800",
      )}
    >
      {React.createElement(resolveIcon(icon), { size: 14 })}
      {label}
    </button>
  );
}

function DetailsView({ details }: { details: Record<string, unknown> }) {
  return (
    <div className="flex flex-col gap-1.5">
      {Object.entries(details).map(([key, value]) => {
        if (isErrorValue(value)) {
          return (
            <div key={key} className="flex items-start gap-2">
              <span className="w-[120px] shrink-0 truncate text-right text-xs text-gray-500" title={key}>
                {key}:
              </span>
              <span className="min-w-0 break-all text-xs font-medium text-red-600">{value.message}</span>
            </div>
          );
        }

        return (
          <div key={key} className="flex items-start gap-2">
            <span className="w-[120px] shrink-0 truncate text-right text-xs text-gray-500" title={key}>
              {key}:
            </span>
            <span className="min-w-0 break-all text-xs text-gray-800">
              {typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "")}
            </span>
          </div>
        );
      })}
    </div>
  );
}
