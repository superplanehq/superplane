/* eslint-disable max-lines-per-function, complexity */
import JsonView from "@uiw/react-json-view";
import { ChevronLeft, ChevronRight, Loader2, X } from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { cn, isUrl, resolveIcon } from "@/lib/utils";
import {
  buildExecutionTabData,
  buildTriggerTabData,
  buildExecutionChain,
  eventBadgeForExecution,
  eventBadgeForTriggeredTrigger,
  getAdjacentRunNodeId,
  getLastRunNodeDetailTab,
  hasObjectValue,
  isErrorValue,
  isRunNodeDetailTabAvailable,
  rememberRunNodeDetailTab,
  resolveRunNodeDetailTab,
  type RunNodeDetailTabKey,
  type RunNodeDetailTabData,
} from "./runNodeDetailModel";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";

export interface RunNodeDetailContentProps {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
  isExecutionsLoading?: boolean;
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
  testId?: string;
}

/** Matches {@link EventSectionDisplay} status chip on canvas nodes (style + casing). */
function EventSectionStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[11px] font-semibold uppercase tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

export function RunNodeDetailContent({
  run,
  nodeId,
  workflowNodes = [],
  componentIconMap = {},
  executions,
  isExecutionsLoading = false,
  onClose,
  onNavigateNode,
  testId = "run-node-detail-content",
}: RunNodeDetailContentProps) {
  const [activeTab, setActiveTab] = useState<RunNodeDetailTabKey>(() => getLastRunNodeDetailTab());
  const previousNodeIdRef = useRef(nodeId);
  const tabSelectionWasFallbackRef = useRef(false);
  const triggerNodeId = run.rootEvent?.nodeId;
  const isTriggerNode = nodeId === triggerNodeId;
  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);
  const previousNodeId = useMemo(() => getAdjacentRunNodeId(executionChain, nodeId, "prev"), [executionChain, nodeId]);
  const nextNodeId = useMemo(() => getAdjacentRunNodeId(executionChain, nodeId, "next"), [executionChain, nodeId]);
  const nodeExecution = useMemo(
    () => executions.find((execution) => execution.nodeId === nodeId),
    [executions, nodeId],
  );
  const workflowNode = useMemo(() => workflowNodes.find((node) => node.id === nodeId), [workflowNodes, nodeId]);

  const nodeName = workflowNode?.name || nodeId;
  const createdAt = isTriggerNode ? run.rootEvent?.createdAt : nodeExecution?.createdAt;

  const tabData = useMemo<RunNodeDetailTabData | null>(() => {
    if (isTriggerNode) {
      return buildTriggerTabData(run, workflowNode);
    }

    if (!nodeExecution) return null;
    return buildExecutionTabData(nodeExecution, workflowNode, workflowNodes);
  }, [isTriggerNode, nodeExecution, run, workflowNode, workflowNodes]);

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
  const headerEventBadge = useMemo(() => {
    if (isTriggerNode) return eventBadgeForTriggeredTrigger(workflowNode);
    if (nodeExecution) return eventBadgeForExecution(workflowNode, nodeExecution);
    return null;
  }, [isTriggerNode, nodeExecution, workflowNode]);
  const hasDetailsSection = hasDetails || !!headerEventBadge || !!createdAt;
  const hasAnyTab = hasDetailsSection || hasPayload || hasConfig;
  const tabAvailability = useMemo(
    () => ({ hasDetailsSection, hasPayload, hasConfig }),
    [hasConfig, hasDetailsSection, hasPayload],
  );

  const selectTab = useCallback((tab: RunNodeDetailTabKey) => {
    rememberRunNodeDetailTab(tab);
    tabSelectionWasFallbackRef.current = false;
    setActiveTab(tab);
  }, []);

  useEffect(() => {
    const nodeChanged = previousNodeIdRef.current !== nodeId;
    previousNodeIdRef.current = nodeId;

    setActiveTab((current) => {
      const preferred = getLastRunNodeDetailTab();
      const resolved = resolveRunNodeDetailTab(preferred, tabAvailability);
      const currentIsValid = isRunNodeDetailTabAvailable(current, tabAvailability);
      const preferredIsValid = isRunNodeDetailTabAvailable(preferred, tabAvailability);

      if (nodeChanged) {
        tabSelectionWasFallbackRef.current = resolved !== preferred;
        return resolved;
      }

      if (currentIsValid) {
        if (tabSelectionWasFallbackRef.current && preferredIsValid) {
          tabSelectionWasFallbackRef.current = false;
          return preferred;
        }
        return current;
      }

      tabSelectionWasFallbackRef.current = resolved !== preferred;
      return resolved;
    });
  }, [nodeId, tabAvailability]);

  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden bg-white"
      data-testid={testId}
      aria-label={`${nodeName} run details`}
    >
      <div className="flex h-9 shrink-0 items-stretch justify-between border-b border-slate-200 pl-3">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <div className="flex min-w-0 items-center gap-1.5">
            <RunNodeIcon
              componentName={workflowNode?.component}
              iconSlug={workflowNode?.component ? componentIconMap[workflowNode.component] : undefined}
              alt={nodeName}
              size={RUN_NODE_ICON_SIZE}
              className="h-3.5 w-3.5 shrink-0 text-gray-800"
            />
            <h3 className="truncate text-[13px] font-medium text-gray-900">{nodeName}</h3>
          </div>
        </div>
        <div className="flex shrink-0 items-stretch">
          {onNavigateNode ? (
            <div className="flex items-center px-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                disabled={!previousNodeId}
                aria-label="Previous node in run"
                onClick={() => previousNodeId && onNavigateNode(previousNodeId)}
              >
                <ChevronLeft className="h-3.5 w-3.5" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
                disabled={!nextNodeId}
                aria-label="Next node in run"
                onClick={() => nextNodeId && onNavigateNode(nextNodeId)}
              >
                <ChevronRight className="h-3.5 w-3.5" />
              </Button>
            </div>
          ) : null}
          <div aria-hidden className="w-px self-stretch bg-slate-200" />
          <div className="flex items-center px-1">
            <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
              <X className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>

      {isExecutionsLoading && !isTriggerNode ? (
        <div className="flex items-center justify-center gap-2 px-4 py-8 text-xs text-gray-400">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading run details...
        </div>
      ) : hasAnyTab ? (
        <>
          <div className="relative z-10 flex h-9 shrink-0 items-stretch overflow-visible border-b border-slate-200 px-2">
            {hasDetailsSection ? (
              <TabButton
                active={activeTab === "details"}
                icon="info"
                label="Details"
                onClick={() => selectTab("details")}
              />
            ) : null}
            {hasPayload ? (
              <TabButton
                active={activeTab === "payload"}
                icon="code"
                label="Payload"
                onClick={() => selectTab("payload")}
              />
            ) : null}
            {hasConfig ? (
              <TabButton
                active={activeTab === "configuration"}
                icon="settings"
                label="Config"
                onClick={() => selectTab("configuration")}
              />
            ) : null}
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
            {activeTab === "details" && hasDetailsSection ? (
              <DetailsView details={tabData?.details ?? {}} statusBadge={headerEventBadge} relativeTime={createdAt} />
            ) : null}
            {activeTab === "payload" && hasPayload ? (
              <JsonView value={tabData?.payload as object} collapsed={2} style={{ fontSize: 12 }} />
            ) : null}
            {activeTab === "configuration" && hasConfig ? (
              <JsonView value={tabData?.configuration as object} collapsed={2} style={{ fontSize: 12 }} />
            ) : null}
          </div>
        </>
      ) : (
        <div className="px-4 py-6 text-center text-xs text-gray-400">No execution data for this node in this run.</div>
      )}
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
        "mb-[-1px] flex items-center gap-1 self-stretch border-b px-2.5 text-[13px] font-medium",
        active ? "border-gray-800 text-gray-800" : "border-transparent text-gray-500 hover:text-gray-800",
      )}
    >
      {React.createElement(resolveIcon(icon), { size: RUN_NODE_ICON_SIZE, className: "h-3.5 w-3.5 shrink-0" })}
      {label}
    </button>
  );
}

function DetailsView({
  details,
  statusBadge,
  relativeTime,
}: {
  details: Record<string, unknown>;
  statusBadge?: { badgeColor: string; label: string } | null;
  relativeTime?: string;
}) {
  return (
    <div className="flex flex-col gap-1.5 text-[13px]">
      {statusBadge ? (
        <div className="flex items-start gap-2">
          <span className="w-[120px] shrink-0 truncate text-right text-gray-500">Status:</span>
          <EventSectionStatusBadge badgeColor={statusBadge.badgeColor} label={statusBadge.label} />
        </div>
      ) : null}
      {relativeTime ? (
        <div className="flex items-start gap-2">
          <span className="w-[120px] shrink-0 truncate text-right text-gray-500">Relative time:</span>
          <span className="min-w-0 break-all text-gray-800">
            <TimeAgo date={relativeTime} />
          </span>
        </div>
      ) : null}
      {Object.entries(details).map(([key, value]) => {
        if (isErrorValue(value)) {
          return (
            <div key={key} className="flex items-start gap-2">
              <span className="w-[120px] shrink-0 truncate text-right text-gray-500" title={key}>
                {key}:
              </span>
              <span className="min-w-0 break-all font-medium text-red-600">{value.message}</span>
            </div>
          );
        }

        return (
          <div key={key} className="flex items-start gap-2">
            <span className="w-[120px] shrink-0 truncate text-right text-gray-500" title={key}>
              {key}:
            </span>
            <DetailValue value={value} />
          </div>
        );
      })}
    </div>
  );
}

function DetailValue({ value }: { value: unknown }) {
  const stringValue = typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "");

  if (isUrl(stringValue)) {
    return (
      <a
        href={stringValue}
        target="_blank"
        rel="noopener noreferrer"
        className="min-w-0 break-all text-blue-600 underline underline-offset-2 hover:text-blue-700"
      >
        {stringValue}
      </a>
    );
  }

  return <span className="min-w-0 break-all text-gray-800">{stringValue}</span>;
}
