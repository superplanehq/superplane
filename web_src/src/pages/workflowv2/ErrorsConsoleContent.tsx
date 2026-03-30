import React, { useMemo } from "react";
import { CircleX } from "lucide-react";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeExecution, ComponentsNode } from "@/api-client";
import { cn, resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { findNode, formatRunTimestamp, getStatusBadgeProps, resolveNodeIconSlug } from "./canvasRunsUtils";
import { buildTriggerSidebarEvent } from "./utils";

function NodeIcon({
  node,
  componentIconMap,
}: {
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
}) {
  const name = node?.name || "Unknown";
  const iconSrc = getHeaderIconSrc(node?.component?.name || node?.trigger?.name);
  const iconSlug = resolveNodeIconSlug(node, componentIconMap);
  if (iconSrc) {
    return <img src={iconSrc} alt={name} className="h-4 w-4 object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "box"), { size: 14, className: "text-gray-500" });
}

function AcknowledgeButton({
  executionId,
  onAcknowledgeErrors,
}: {
  executionId: string;
  onAcknowledgeErrors: (ids: string[]) => void;
}) {
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onAcknowledgeErrors([executionId]);
      }}
      className="rounded px-1.5 py-0.5 text-[10px] font-medium text-gray-500 hover:bg-gray-200 hover:text-gray-700 transition-colors whitespace-nowrap"
    >
      Acknowledge
    </button>
  );
}

function ErrorItemRow({
  item,
  componentIconMap,
  onNodeSelect,
  onExecutionSelect,
  onAcknowledgeErrors,
}: {
  item: {
    execution: CanvasesCanvasNodeExecution;
    event: CanvasesCanvasEventWithExecutions;
    node: ComponentsNode | undefined;
    triggerNode: ComponentsNode | undefined;
  };
  componentIconMap: Record<string, string>;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
}) {
  const nodeName = item.node?.name || item.execution.nodeId || "Unknown";
  const { badgeColor, label } = getStatusBadgeProps("error");
  const triggerSidebarEvent = buildTriggerSidebarEvent(item.event, item.triggerNode);

  const handleSelect = () => {
    if (onExecutionSelect && item.execution.id && item.event.id && item.execution.nodeId) {
      onExecutionSelect({
        nodeId: item.execution.nodeId,
        eventId: item.event.id,
        executionId: item.execution.id,
        triggerEvent: triggerSidebarEvent,
      });
    } else if (onNodeSelect && item.execution.nodeId) {
      onNodeSelect(item.execution.nodeId);
    }
  };

  const isClickable = onNodeSelect || onExecutionSelect;

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 min-h-8",
        isClickable && "cursor-pointer hover:bg-gray-50 transition-colors",
      )}
      role={isClickable ? "button" : undefined}
      tabIndex={isClickable ? 0 : undefined}
      onClick={handleSelect}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleSelect();
        }
      }}
    >
      <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
        <NodeIcon node={item.node} componentIconMap={componentIconMap} />
      </div>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        <span className="text-xs text-gray-700 truncate">{nodeName}</span>
        <div
          className={`uppercase text-[10px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${badgeColor}`}
        >
          <span>{label}</span>
        </div>
        {item.execution.resultMessage && (
          <span className="text-xs text-red-600 truncate max-w-[300px]">{item.execution.resultMessage}</span>
        )}
      </div>
      {onAcknowledgeErrors && item.execution.id && (
        <AcknowledgeButton executionId={item.execution.id} onAcknowledgeErrors={onAcknowledgeErrors} />
      )}
      <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">
        {item.execution.createdAt ? formatRunTimestamp(item.execution.createdAt) : ""}
      </span>
    </div>
  );
}

export function ErrorsConsoleContent({
  events,
  nodes,
  componentIconMap = {},
  searchQuery,
  onNodeSelect,
  onExecutionSelect,
  onAcknowledgeErrors,
}: {
  events: CanvasesCanvasEventWithExecutions[];
  nodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  searchQuery: string;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
}) {
  const errorItems = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    const items: {
      execution: CanvasesCanvasNodeExecution;
      event: CanvasesCanvasEventWithExecutions;
      node: ComponentsNode | undefined;
      triggerNode: ComponentsNode | undefined;
    }[] = [];

    for (const event of events) {
      const triggerNode = nodes.find((n) => n.id === event.nodeId);
      for (const exec of event.executions || []) {
        if (exec.result !== "RESULT_FAILED") continue;
        if (exec.resultReason === "RESULT_REASON_ERROR_RESOLVED") continue;

        const node = findNode(nodes, exec.nodeId);

        if (query) {
          const searchable = [node?.name, exec.nodeId, exec.resultMessage, event.id]
            .filter(Boolean)
            .join(" ")
            .toLowerCase();
          if (!searchable.includes(query)) continue;
        }

        items.push({ execution: exec, event, node, triggerNode });
      }
    }
    return items;
  }, [events, nodes, searchQuery]);

  const allErrorIds = useMemo(() => errorItems.map((item) => item.execution.id!).filter(Boolean), [errorItems]);

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="flex items-center gap-2 px-4 py-1.5 border-b border-gray-200">
        <span className="text-[11px] font-medium text-gray-600">
          {errorItems.length} unacknowledged {errorItems.length === 1 ? "error" : "errors"}
        </span>
        {onAcknowledgeErrors && allErrorIds.length > 0 && (
          <button
            type="button"
            onClick={() => onAcknowledgeErrors(allErrorIds)}
            className="ml-auto rounded-md px-2 py-0.5 text-[11px] font-medium text-gray-600 hover:bg-gray-100 transition-colors"
          >
            Acknowledge all
          </button>
        )}
      </div>
      <div className="flex-1 overflow-auto">
        {errorItems.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-4 py-10 text-center">
            <CircleX className="h-6 w-6 text-gray-300 mb-2" />
            <p className="text-[13px] font-medium text-gray-600">No unacknowledged errors</p>
            <p className="mt-0.5 text-xs text-gray-400">All errors have been acknowledged.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {errorItems.map((item) => (
              <ErrorItemRow
                key={item.execution.id}
                item={item}
                componentIconMap={componentIconMap}
                onNodeSelect={onNodeSelect}
                onExecutionSelect={onExecutionSelect}
                onAcknowledgeErrors={onAcknowledgeErrors}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
