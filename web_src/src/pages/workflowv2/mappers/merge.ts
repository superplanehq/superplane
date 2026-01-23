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

  // Check for timeout - timeout results in RESULT_FAILED with "timed out" message
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
    if (execution.resultMessage?.toLowerCase().includes("timed out")) {
      return "timeout";
    }

    // Failed state - stopIfExpression triggered or other error
    return "failed";
  }

  // Success state - merge completed normally
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
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
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
    additionalData?: unknown,
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    return {
      iconSlug: componentDefinition?.icon || "git-merge",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      collapsedBackground: getBackgroundColorClass("white"),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition?.label || "Merge",
      eventSections: lastExecution
        ? getMergeEventSections(nodes, lastExecution, nodeQueueItems, additionalData)
        : getQueueOnlyEventSections(nodes, nodeQueueItems),
      includeEmptyState: !lastExecution && (!nodeQueueItems || nodeQueueItems.length === 0),
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
  nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
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

  // Add queue section if there are queued items
  if (nodeQueueItems && nodeQueueItems.length > 0) {
    const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
    const queueRootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
    const queueRootTriggerRenderer = getTriggerRenderer(queueRootTriggerNode?.trigger?.name || "");

    if (queueItem.rootEvent) {
      const { title } = queueRootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
      sections.push({
        receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
        eventTitle: title,
        eventState: "queued" as const,
      });
    }
  }

  return sections;
}

function getQueueOnlyEventSections(
  nodes: ComponentsNode[],
  nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
): EventSection[] | undefined {
  if (!nodeQueueItems || nodeQueueItems.length === 0) {
    return undefined;
  }

  const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
  const queueRootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
  const queueRootTriggerRenderer = getTriggerRenderer(queueRootTriggerNode?.trigger?.name || "");

  if (queueItem.rootEvent) {
    const { title } = queueRootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
    return [
      {
        receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
        eventTitle: title,
        eventState: "queued" as const,
      },
    ];
  }

  return undefined;
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
