import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, StateFunction } from "./types";
import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";

// Output channel names matching backend
const CHANNEL_SUCCESS = "success";
const CHANNEL_TIMEOUT = "timeout";
const CHANNEL_FAIL = "fail";

/**
 * Output payload type
 */
interface OutputPayload {
  type: string;
  timestamp: string;
  data: any;
}

/**
 * Type for merge outputs with channel structure
 */
type MergeOutputs = {
  success?: OutputPayload[];
  timeout?: OutputPayload[];
  fail?: OutputPayload[];
  default?: OutputPayload[]; // For backwards compatibility
};

/**
 * Metadata structure for merge execution (from backend)
 */
interface MergeExecutionMetadata {
  groupKey?: string;
  eventIDs?: string[];
  sources?: string[];
  stopEarly?: boolean;
}

/**
 * Additional data passed to merge mapper
 */
interface MergeAdditionalData {
  incomingSourcesCount?: number;
}

/**
 * Determines which output channel has data, indicating the merge outcome.
 * Returns the channel name or null if no output found.
 */
function getActiveChannel(execution: WorkflowsWorkflowNodeExecution): string | null {
  const outputs = execution.outputs as MergeOutputs | undefined;
  if (!outputs) return null;

  if (outputs.success && outputs.success.length > 0) return CHANNEL_SUCCESS;
  if (outputs.timeout && outputs.timeout.length > 0) return CHANNEL_TIMEOUT;
  if (outputs.fail && outputs.fail.length > 0) return CHANNEL_FAIL;
  if (outputs.default && outputs.default.length > 0) return "default";

  return null;
}

export const MERGE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-orange-100 dark:bg-orange-900/50",
    badgeColor: "bg-yellow-600",
  },
  success: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
  timeout: {
    icon: "clock",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
};

/**
 * Merge-specific state logic function
 */
export const mergeStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
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
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
    additionalData?: unknown,
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    return {
      iconSlug: componentDefinition?.icon || "git-merge",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      collapsedBackground: getBackgroundColorClass("white"),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition?.label || "Merge",
      eventSections: lastExecution ? getMergeEventSections(nodes, lastExecution, additionalData) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: MERGE_STATE_MAP,
    };
  },

  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution, additionalData?: unknown): string {
    return getMergeSubtitle(execution, additionalData);
  },

  getExecutionDetails(
    execution: WorkflowsWorkflowNodeExecution,
    _node: ComponentsNode,
    nodes?: ComponentsNode[],
  ): Record<string, any> {
    const details: Record<string, any> = {};
    const metadata = execution.metadata as MergeExecutionMetadata | undefined;

    if (execution.createdAt) {
      details["Started at"] = new Date(execution.createdAt).toLocaleString();
    }

    if (execution.state === "STATE_FINISHED" && execution.updatedAt) {
      details["Finished at"] = new Date(execution.updatedAt).toLocaleString();
    }

    // Build timeline for events received
    if (metadata?.sources && metadata.sources.length > 0) {
      details["Events"] = buildMergeTimeline(metadata, execution, nodes);
    }

    return details;
  },
};

function getMergeEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  additionalData?: unknown,
): EventSection[] {
  const sections: EventSection[] = [];

  // Add the main execution section
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSubtitle = getMergeSubtitle(execution, additionalData);

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: eventTitle,
    eventSubtitle: eventSubtitle,
    eventState: mergeStateFunction(execution),
    eventId: execution.rootEvent?.id,
  });

  return sections;
}

function getMergeSubtitle(execution: WorkflowsWorkflowNodeExecution, additionalData?: unknown): string {
  const metadata = execution.metadata as MergeExecutionMetadata | undefined;
  const mergeData = additionalData as MergeAdditionalData | undefined;

  const sourcesReceived = metadata?.sources?.length || 0;
  const sourcesNeeded = mergeData?.incomingSourcesCount;

  // Determine timestamp - use updatedAt for finished, createdAt otherwise
  const timestamp =
    execution.state === "STATE_FINISHED" && execution.updatedAt ? execution.updatedAt : execution.createdAt;

  const timeAgo = timestamp ? formatTimeAgo(new Date(timestamp)) : "";

  // For waiting state, show progress
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    if (sourcesNeeded !== undefined && sourcesNeeded > 0) {
      return `${sourcesReceived}/${sourcesNeeded} received · ${timeAgo}`;
    }
    // If we don't know the needed count, just show received
    return `${sourcesReceived} received · ${timeAgo}`;
  }

  // For completed states, show the final count and time
  if (execution.state === "STATE_FINISHED") {
    if (sourcesNeeded !== undefined && sourcesNeeded > 0) {
      return `${sourcesReceived}/${sourcesNeeded} received · ${timeAgo}`;
    }
    return timeAgo;
  }

  return timeAgo;
}

/**
 * Timeline entry type for merge events (matches ApprovalTimelineEntry format)
 */
interface MergeTimelineEntry {
  label: string;
  status: string;
  timestamp?: string;
  comment?: string;
}

/**
 * Build a timeline showing received events for the merge component
 */
function buildMergeTimeline(
  metadata: MergeExecutionMetadata,
  execution: WorkflowsWorkflowNodeExecution,
  nodes?: ComponentsNode[],
): MergeTimelineEntry[] {
  const timeline: MergeTimelineEntry[] = [];

  // Add an entry for each source that contributed
  if (metadata.sources) {
    metadata.sources.forEach((sourceId) => {
      // Look up the source node to get its label/name
      const sourceNode = nodes?.find((n) => n.id === sourceId);
      const sourceLabel = sourceNode?.name || sourceNode?.component?.name || `Node ${sourceId.substring(0, 8)}...`;

      // Format timestamp in "ago" style
      const timestampFormatted = execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : undefined;

      timeline.push({
        label: sourceLabel,
        status: "Received",
        timestamp: timestampFormatted,
      });
    });
  }

  // If merge stopped early, add that to the timeline
  if (metadata.stopEarly) {
    const stoppedAtFormatted = execution.updatedAt ? formatTimeAgo(new Date(execution.updatedAt)) : undefined;
    timeline.push({
      label: "Condition met",
      status: "Stopped",
      timestamp: stoppedAtFormatted,
      comment: "Merge completed early",
    });
  }

  return timeline;
}
