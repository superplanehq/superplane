import React, { useMemo } from "react";
import { CircleX } from "lucide-react";
import type {
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn, resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { findNode, getStatusBadgeProps, resolveNodeIconSlug } from "@/pages/app/lib/canvas-runs";

function NodeIcon({
  node,
  componentIconMap,
}: {
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
}) {
  const name = node?.name || "Unknown";
  const iconSrc = getHeaderIconSrc(node?.component);
  const iconSlug = resolveNodeIconSlug(node, componentIconMap);
  if (iconSrc) {
    return <img src={iconSrc} alt={name} className="h-4 w-4 object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "box"), {
    size: 14,
    className: "text-gray-500 dark:text-gray-400",
  });
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
      className="rounded px-1.5 py-0.5 text-[10px] font-medium text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100 transition-colors whitespace-nowrap"
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
    execution: CanvasesCanvasNodeExecutionRef;
    run: CanvasesCanvasRun;
    rootEventId: string;
    node: ComponentsNode | undefined;
  };
  componentIconMap: Record<string, string>;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: { runId: string; nodeId: string }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
}) {
  const nodeName = item.node?.name || item.execution.nodeId || "Unknown";
  const { badgeColor, label } = getStatusBadgeProps("error");

  const handleSelect = () => {
    if (onExecutionSelect && item.run.id && item.execution.nodeId) {
      onExecutionSelect({
        runId: item.run.id,
        nodeId: item.execution.nodeId,
      });
    } else if (onNodeSelect && item.execution.nodeId) {
      onNodeSelect(item.execution.nodeId);
    }
  };

  const isClickable = onNodeSelect || onExecutionSelect;

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 min-h-8 text-gray-800 dark:text-gray-100",
        isClickable && "cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors",
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
        <span className="text-xs text-gray-700 dark:text-gray-200 truncate">{nodeName}</span>
        <div
          className={cn(
            "uppercase text-[10px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white",
            withEventStatusBadgeClasses(badgeColor),
          )}
        >
          <span>{label}</span>
        </div>
        {item.execution.resultMessage && (
          <span className="text-xs text-red-600 dark:text-red-400 truncate max-w-[300px]">
            {item.execution.resultMessage}
          </span>
        )}
      </div>
      {onAcknowledgeErrors && item.execution.id && (
        <AcknowledgeButton executionId={item.execution.id} onAcknowledgeErrors={onAcknowledgeErrors} />
      )}
      <span className="text-xs text-gray-400 dark:text-gray-500 tabular-nums whitespace-nowrap">
        {item.execution.createdAt ? (
          <Timestamp date={item.execution.createdAt} display="relative" relativeStyle="abbreviated" />
        ) : (
          ""
        )}
      </span>
    </div>
  );
}

export function ErrorsConsoleContent({
  runs,
  nodes,
  componentIconMap = {},
  searchQuery,
  onNodeSelect,
  onExecutionSelect,
  onAcknowledgeErrors,
}: {
  runs: CanvasesCanvasRun[];
  nodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  searchQuery: string;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: { runId: string; nodeId: string }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
}) {
  const errorItems = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    const items: {
      execution: CanvasesCanvasNodeExecutionRef;
      run: CanvasesCanvasRun;
      rootEventId: string;
      node: ComponentsNode | undefined;
    }[] = [];

    for (const run of runs) {
      const rootEventId = run.rootEvent?.id;
      if (!rootEventId) continue;

      for (const exec of run.executions || []) {
        if (exec.result !== "RESULT_FAILED") continue;
        if (exec.resultReason === "RESULT_REASON_ERROR_RESOLVED") continue;

        const node = findNode(nodes, exec.nodeId);

        if (query) {
          const searchable = [node?.name, exec.nodeId, exec.resultMessage, rootEventId]
            .filter(Boolean)
            .join(" ")
            .toLowerCase();
          if (!searchable.includes(query)) continue;
        }

        items.push({ execution: exec, run, rootEventId, node });
      }
    }
    return items;
  }, [runs, nodes, searchQuery]);

  const allErrorIds = useMemo(() => errorItems.map((item) => item.execution.id!).filter(Boolean), [errorItems]);

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className={cn("flex items-center gap-2 px-4 py-1.5 border-b", appDarkModeClasses.sidebarEdge)}>
        <span className="text-[11px] font-medium text-gray-600 dark:text-gray-400">
          {errorItems.length} unacknowledged {errorItems.length === 1 ? "error" : "errors"}
        </span>
        {onAcknowledgeErrors && allErrorIds.length > 0 && (
          <button
            type="button"
            onClick={() => onAcknowledgeErrors(allErrorIds)}
            className="ml-auto rounded-md px-2 py-0.5 text-[11px] font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100 transition-colors"
          >
            Acknowledge all
          </button>
        )}
      </div>
      <div className="flex-1 overflow-auto">
        {errorItems.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-4 py-10 text-center">
            <CircleX className="h-6 w-6 text-gray-300 dark:text-gray-600 mb-2" />
            <p className="text-[13px] font-medium text-gray-600 dark:text-gray-300">No unacknowledged errors</p>
            <p className="mt-0.5 text-xs text-gray-400 dark:text-gray-500">All errors have been acknowledged.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-800/50">
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
