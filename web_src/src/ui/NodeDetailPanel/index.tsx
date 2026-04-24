import React, { useEffect, useMemo, useState } from "react";
import JsonView from "@uiw/react-json-view";
import { ChevronLeft, ChevronRight, X } from "lucide-react";

import type {
  CanvasesCanvasNodeExecution,
  CanvasesDescribeRunResponse,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { TimeAgo } from "@/components/TimeAgo";
import { cn, flattenObject, resolveIcon } from "@/lib/utils";
import { getExecutionDetails } from "@/pages/workflowv2/mappers";
import {
  badgeColorForEventState,
  getStatusBadgeProps,
  resolveExecutionBadgeColor,
  resolveExecutionDisplayStatus,
  resolveExecutionEventState,
} from "@/pages/workflowv2/lib/canvas-runs";

interface NodeDetailPanelProps {
  nodeId: string;
  runData: CanvasesDescribeRunResponse;
  workflowNodes?: ComponentsNode[];
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
}

type TabKey = "current" | "payload" | "configuration";

//
// Unified status badge styling used in the node detail header. Colors are
// resolved the same way as the StepRibbon and Activity rows so component-
// specific labels (e.g. "False" for an IF evaluating falsey, "Pushed Through"
// for a wait) show the right palette instead of defaulting to a generic
// "Passed" pill.
//
function StatusPill({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded px-1.5 py-[1px] text-[10px] font-semibold uppercase tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

function ExecutionStatusBadge({
  execution,
  workflowNodes,
}: {
  execution: CanvasesCanvasNodeExecution;
  workflowNodes: ComponentsNode[];
}) {
  const rawStatus = resolveExecutionDisplayStatus(execution, workflowNodes);
  const eventState = resolveExecutionEventState(execution);
  const badgeColor = resolveExecutionBadgeColor(execution, workflowNodes);
  const { label, badgeColor: resolvedColor } = getStatusBadgeProps(rawStatus, eventState, badgeColor);
  return <StatusPill badgeColor={resolvedColor} label={label} />;
}

function TriggerStatusBadge() {
  const { label, badgeColor } = getStatusBadgeProps("triggered", "triggered", badgeColorForEventState("triggered"));
  return <StatusPill badgeColor={badgeColor} label={label} />;
}

type TabData = {
  current?: Record<string, unknown>;
  payload?: unknown;
  configuration?: unknown;
};

function buildExecutionTabData(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: ComponentsNode | undefined,
  workflowNodes: ComponentsNode[],
): TabData {
  const tabData: TabData = {};
  let currentData: Record<string, unknown> = {};

  if (workflowNode?.component?.name) {
    const customDetails = getExecutionDetails(workflowNode.component.name, execution, workflowNode, workflowNodes);
    if (customDetails && Object.keys(customDetails).length > 0) {
      currentData = { ...customDetails };
    }
  }

  if (Object.keys(currentData).length === 0) {
    const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
    const dataSource = (hasOutputs ? execution.outputs : execution.metadata) || {};
    currentData = { ...flattenObject(dataSource) };
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED") &&
    !("Error" in currentData)
  ) {
    currentData["Error"] = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  if (execution.result === "RESULT_CANCELLED" && !("Cancelled by" in currentData)) {
    const cancelledBy = execution.cancelledBy;
    currentData["Cancelled by"] = cancelledBy?.name || cancelledBy?.id || "Unknown";
  }

  tabData.current = Object.fromEntries(
    Object.entries(currentData).filter(([, value]) => value !== undefined && value !== "" && value !== null),
  );

  let payload: Record<string, unknown> = {};
  if (execution.outputs) {
    const outputData = Object.values(execution.outputs).find((output) => Array.isArray(output) && output.length > 0) as
      | unknown[]
      | undefined;
    if (outputData && outputData.length > 0) {
      payload = outputData[0] as Record<string, unknown>;
    }
  }
  tabData.payload = payload;

  if (execution.configuration && Object.keys(execution.configuration).length > 0) {
    tabData.configuration = execution.configuration;
  }

  return tabData;
}

function buildTriggerTabData(runEvent: CanvasesDescribeRunResponse["run"]): TabData {
  const tabData: TabData = {};

  const currentData: Record<string, unknown> = {};
  if (runEvent?.channel) currentData["Channel"] = runEvent.channel;
  if (runEvent?.customName) currentData["Name"] = runEvent.customName;
  if (runEvent?.createdAt) currentData["Triggered at"] = runEvent.createdAt;

  tabData.current = Object.keys(currentData).length > 0 ? currentData : undefined;
  tabData.payload = runEvent?.data && Object.keys(runEvent.data).length > 0 ? runEvent.data : undefined;

  return tabData;
}

function isErrorValue(value: unknown): value is { __type: "error"; message: string } {
  return !!value && typeof value === "object" && (value as { __type?: string }).__type === "error";
}

export function NodeDetailPanel({ nodeId, runData, workflowNodes, onClose, onNavigateNode }: NodeDetailPanelProps) {
  const [activeTab, setActiveTab] = useState<TabKey>("current");
  const executions = runData.executions || [];
  const triggerNodeId = runData.run?.nodeId;

  const nodeExecution = useMemo(() => executions.find((e) => e.nodeId === nodeId), [executions, nodeId]);
  const isTriggerNode = nodeId === triggerNodeId;

  const executionChain = useMemo(() => {
    const chain: string[] = [];
    const visited = new Set<string>();

    if (triggerNodeId) {
      chain.push(triggerNodeId);
      visited.add(triggerNodeId);
    }

    for (const exec of executions) {
      if (exec.nodeId && !visited.has(exec.nodeId)) {
        visited.add(exec.nodeId);
        chain.push(exec.nodeId);
      }
    }
    return chain;
  }, [executions, triggerNodeId]);

  const currentIndex = executionChain.indexOf(nodeId);
  const prevNodeId = currentIndex > 0 ? executionChain[currentIndex - 1] : null;
  const nextNodeId =
    currentIndex >= 0 && currentIndex < executionChain.length - 1 ? executionChain[currentIndex + 1] : null;

  const workflowNode = useMemo(() => workflowNodes?.find((n) => n.id === nodeId), [workflowNodes, nodeId]);
  const nodeName = workflowNode?.name || nodeId;

  const tabData = useMemo<TabData | null>(() => {
    if (isTriggerNode) {
      return buildTriggerTabData(runData.run);
    }
    if (!nodeExecution) return null;
    return buildExecutionTabData(nodeExecution, workflowNode, workflowNodes || []);
  }, [isTriggerNode, runData.run, nodeExecution, workflowNode, workflowNodes]);

  useEffect(() => {
    setActiveTab("current");
  }, [nodeId]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const hasDetails = tabData?.current && Object.keys(tabData.current).length > 0;
  const hasPayload =
    !!tabData?.payload && typeof tabData.payload === "object" && Object.keys(tabData.payload as object).length > 0;
  const hasConfig =
    !!tabData?.configuration &&
    typeof tabData.configuration === "object" &&
    Object.keys(tabData.configuration as object).length > 0;
  const hasAnyTab = hasDetails || hasPayload || hasConfig;

  const createdAt = isTriggerNode ? runData.run?.createdAt : nodeExecution?.createdAt;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="flex w-full max-w-3xl max-h-[80vh] flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow-2xl"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-1.5">
        <div className="flex items-center gap-3 min-w-0">
          {onNavigateNode ? (
            <div className="flex items-center gap-0.5">
              <Button
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                onClick={() => prevNodeId && onNavigateNode(prevNodeId)}
                disabled={!prevNodeId}
              >
                <ChevronLeft className="h-3.5 w-3.5" />
              </Button>
              <span className="text-[11px] text-gray-400 tabular-nums">
                {currentIndex >= 0 ? `${currentIndex + 1}/${executionChain.length}` : ""}
              </span>
              <Button
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
            <TriggerStatusBadge />
          ) : nodeExecution ? (
            <ExecutionStatusBadge execution={nodeExecution} workflowNodes={workflowNodes || []} />
          ) : null}
          {createdAt ? (
            <span className="text-xs text-gray-400">
              <TimeAgo date={createdAt} />
            </span>
          ) : null}
        </div>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {hasAnyTab ? (
        <>
          <div className="flex shrink-0 items-center h-8 border-b border-slate-200 px-2">
            {hasDetails ? (
              <button
                type="button"
                onClick={() => setActiveTab("current")}
                className={cn(
                  "flex items-center gap-1 px-2.5 py-1.5 text-[13px] font-medium border-b",
                  activeTab === "current"
                    ? "text-gray-800 border-gray-800"
                    : "text-gray-500 hover:text-gray-800 border-transparent",
                )}
              >
                {React.createElement(resolveIcon("info"), { size: 14 })}
                Details
              </button>
            ) : null}
            {hasPayload ? (
              <button
                type="button"
                onClick={() => setActiveTab("payload")}
                className={cn(
                  "flex items-center gap-1 px-2.5 py-1.5 text-[13px] font-medium border-b",
                  activeTab === "payload"
                    ? "text-gray-800 border-gray-800"
                    : "text-gray-500 hover:text-gray-800 border-transparent",
                )}
              >
                {React.createElement(resolveIcon("code"), { size: 14 })}
                Payload
              </button>
            ) : null}
            {hasConfig ? (
              <button
                type="button"
                onClick={() => setActiveTab("configuration")}
                className={cn(
                  "flex items-center gap-1 px-2.5 py-1.5 text-[13px] font-medium border-b",
                  activeTab === "configuration"
                    ? "text-gray-800 border-gray-800"
                    : "text-gray-500 hover:text-gray-800 border-transparent",
                )}
              >
                {React.createElement(resolveIcon("settings"), { size: 14 })}
                Config
              </button>
            ) : null}
          </div>

          <div className="flex-1 overflow-y-auto px-4 py-3">
            {activeTab === "current" && tabData?.current ? (
              <div className="flex flex-col gap-1.5">
                {Object.entries(tabData.current).map(([key, value]) => {
                  if (isErrorValue(value)) {
                    return (
                      <div key={key} className="flex items-start gap-2">
                        <span className="shrink-0 text-xs text-gray-500 w-[120px] text-right truncate" title={key}>
                          {key}:
                        </span>
                        <span className="min-w-0 break-all text-xs text-red-600 font-medium">{value.message}</span>
                      </div>
                    );
                  }
                  return (
                    <div key={key} className="flex items-start gap-2">
                      <span className="shrink-0 text-xs text-gray-500 w-[120px] text-right truncate" title={key}>
                        {key}:
                      </span>
                      <span className="min-w-0 break-all text-xs text-gray-800">
                        {typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "")}
                      </span>
                    </div>
                  );
                })}
              </div>
            ) : null}

            {activeTab === "payload" && tabData?.payload ? (
              <JsonView value={tabData.payload as object} collapsed={2} style={{ fontSize: 12 }} />
            ) : null}

            {activeTab === "configuration" && tabData?.configuration ? (
              <JsonView value={tabData.configuration as object} collapsed={2} style={{ fontSize: 12 }} />
            ) : null}
          </div>
        </>
      ) : (
        <div className="px-4 py-6 text-center text-xs text-gray-400">No execution data for this node in this run.</div>
      )}
      </div>
    </div>
  );
}
