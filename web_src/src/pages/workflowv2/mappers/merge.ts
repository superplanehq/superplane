import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  StateFunction,
  SubtitleContext,
  OutputPayload,
} from "./types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { truncate } from "./safeMappers";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderWithTimeAgo } from "@/components/TimeAgo";
import { formatRelativeTime, formatTimestampInUserTimezone } from "@/lib/timezone";

// Output channel names matching backend
const CHANNEL_SUCCESS = "success";
const CHANNEL_TIMEOUT = "timeout";
const CHANNEL_FAIL = "fail";

/**
 * Type for merge outputs with channel structure
 */
type Outputs = {
  success?: OutputPayload[];
  timeout?: OutputPayload[];
  fail?: OutputPayload[];
};

/**
 * Metadata structure for merge execution (from backend)
 */
interface ExecutionMetadata {
  groupKey?: string;
  eventIDs?: string[];
  sourceNodes?: SourceNode[];
  stopEarly?: boolean;
}

interface SourceNode {
  nodeId: string;
  receivedAt?: string;
}

/**
 * Determines which output channel has data, indicating the merge outcome.
 * Returns the channel name or null if no output found.
 */
function getActiveChannel(execution: ExecutionInfo): string | null {
  const outputs = execution.outputs as Outputs | undefined;
  if (!outputs) return null;

  if (outputs.success && outputs.success.length > 0) return CHANNEL_SUCCESS;
  if (outputs.timeout && outputs.timeout.length > 0) return CHANNEL_TIMEOUT;
  if (outputs.fail && outputs.fail.length > 0) return CHANNEL_FAIL;

  return null;
}

export const MERGE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  success: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  timeout: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

/**
 * Merge-specific state logic function
 */
export const mergeStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  // Check for cancellation
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // Waiting state - merge is still collecting events
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "waiting";
  }

  // For finished executions, determine state from active output channel
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const activeChannel = getActiveChannel(execution);

    if (activeChannel === CHANNEL_SUCCESS) return "success";
    if (activeChannel === CHANNEL_TIMEOUT) return "timeout";
    if (activeChannel === CHANNEL_FAIL) return "failed";

    // Backwards compatibility for legacy executions using default channel
    if (activeChannel === "default") return "success";

    // No output found - default to success for finished/passed
    return "success";
  }

  // Handle error states (actual failures, not routed to fail channel)
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
    return "failed";
  }

  return "neutral";
};

/**
 * Merge-specific state registry
 */
export const MERGE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: MERGE_STATE_MAP,
  getState: mergeStateFunction,
};

export const mergeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      iconSlug: context.componentDefinition?.icon || "git-merge",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsedBackground: getBackgroundColorClass("white"),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Merge",
      eventSections: lastExecution
        ? getMergeEventSections(context.nodes, lastExecution, context.additionalData)
        : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: MERGE_STATE_MAP,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return getMergeSubtitle(context.execution, context.additionalData);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};
    const metadata = context.execution.metadata as ExecutionMetadata | undefined;

    if (context.execution.createdAt) {
      details["Started at"] = formatTimestampInUserTimezone(context.execution.createdAt);
    }

    if (context.execution.state === "STATE_FINISHED" && context.execution.updatedAt) {
      details["Finished at"] = formatTimestampInUserTimezone(context.execution.updatedAt);
    }

    return withSources(details, metadata, context.nodes);
  },
};

function getMergeEventSections(nodes: NodeInfo[], execution: ExecutionInfo, additionalData?: unknown): EventSection[] {
  const sections: EventSection[] = [];

  // Add the main execution section
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  const eventSubtitle = getMergeSubtitle(execution, additionalData);

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: eventTitle,
    eventSubtitle: eventSubtitle,
    eventState: mergeStateFunction(execution),
    eventId: execution.rootEvent!.id!,
  });

  return sections;
}

type SourceSummary = {
  expected: number;
  received: number;
};

function sourceSummary(metadata: ExecutionMetadata | undefined): SourceSummary {
  const summary: SourceSummary = {
    expected: 0,
    received: 0,
  };

  if (metadata?.sourceNodes) {
    summary.expected = metadata.sourceNodes.length;
    summary.received = metadata.sourceNodes.filter((s) => s.receivedAt).length;
  }

  return summary;
}

function getMergeSubtitle(execution: ExecutionInfo, _: unknown): string | React.ReactNode {
  const metadata = execution.metadata as ExecutionMetadata | undefined;
  const summary = sourceSummary(metadata);

  // For waiting state, show progress
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    const prefix = `${summary.received}/${summary.expected} received`;
    return renderWithTimeAgo(prefix, execution.createdAt!);
  }

  return renderWithTimeAgo(`${summary.received}/${summary.expected} received`, execution.updatedAt!);
}

function withSources(
  details: Record<string, string>,
  metadata: ExecutionMetadata | undefined,
  nodes: NodeInfo[],
): Record<string, string> {
  if (!metadata?.sourceNodes) {
    return details;
  }

  const received = metadata.sourceNodes
    .filter((s) => s.receivedAt)
    .sort((a, b) => new Date(a.receivedAt!).getTime() - new Date(b.receivedAt!).getTime());

  const unreceived = metadata.sourceNodes.filter((s) => !s.receivedAt);

  //
  // Add the received sources first.
  //
  for (const source of received) {
    const sourceNode = nodes.find((n) => n.id === source.nodeId);
    const sourceLabel = sourceNode?.name || sourceNode?.componentName || `Node ${truncate(source.nodeId, 8)}`;
    details[sourceLabel] = `Received ${formatRelativeTime(source.receivedAt!, true)}`;
  }

  //
  // Add the unreceived sources last.
  //
  for (const source of unreceived) {
    const sourceNode = nodes?.find((n) => n.id === source.nodeId);
    const sourceLabel = sourceNode?.name || sourceNode?.componentName || `Node ${truncate(source.nodeId, 8)}`;
    details[sourceLabel] = source.receivedAt ? formatRelativeTime(source.receivedAt, true) : "-";
  }

  return details;
}
