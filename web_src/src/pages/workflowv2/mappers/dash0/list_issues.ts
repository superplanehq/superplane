import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection, ComponentBaseSpec, EventState, EventStateMap, DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload, EventStateRegistry, StateFunction } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { ListIssuesConfiguration, PrometheusResponse } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const listIssuesMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    const configuration = node.configuration as unknown as ListIssuesConfiguration;
    const specs = getSpecs(configuration);
    
    return {
      iconSrc: dash0Icon,
      iconBackground: "bg-white",
      headerColor: "bg-white",
      collapsedBackground: "bg-white",
      collapsed: node.isCollapsed,
      title: node.name!,
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      specs,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(
    _node: ComponentsNode,
    execution: WorkflowsWorkflowNodeExecution,
    additionalData?: unknown,
  ): string {
    // Check if this is being called from ChainItem (which passes additionalData as undefined or a different structure)
    // For ChainItem, just return the time without counts
    const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
    
    // If additionalData is explicitly a marker object indicating ChainItem context, skip counts
    // Otherwise, include counts for SidebarEventItem
    if (additionalData && typeof additionalData === 'object' && 'skipIssueCounts' in additionalData) {
      return timeAgo;
    }

    const { critical, degraded } = getIssueCounts(execution);

    // Build subtitle with counts and time
    const countParts: string[] = [];
    if (critical > 0) {
      countParts.push(`${critical} critical`);
    }
    if (degraded > 0) {
      countParts.push(`${degraded} degraded`);
    }

    if (countParts.length > 0) {
      return `${countParts.join(", ")} · ${timeAgo}`;
    }

    return timeAgo;
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Issues: "No issues found" };
    }

    const responseData = outputs.default[0]?.data as Record<string, any> | undefined;

    if (!responseData) {
      return { Issues: "No issues found" };
    }

    // Format the issues response data for display
    const details: Record<string, string> = {};
    try {
      const formatted = JSON.stringify(responseData, null, 2);
      details["Issues Data"] = formatted;
    } catch (error) {
      details["Issues Data"] = String(responseData);
    }

    return details;
  },
};

function metadataList(_node: ComponentsNode): MetadataItem[] {
  return [];
}

function getSpecs(configuration: ListIssuesConfiguration): ComponentBaseSpec[] | undefined {
  if (!configuration?.checkRules || configuration.checkRules.length === 0) {
    return undefined;
  }

  return [
    {
      title: "Check Rule",
      tooltipTitle: "Check Rules",
      values: configuration.checkRules.map((rule) => ({
        badges: [
          {
            label: rule,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    },
  ];
}

export const LIST_ISSUES_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  clear: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  degraded: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-yellow-100",
    badgeColor: "bg-yellow-500",
  },
  critical: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const listIssuesStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  // Handle error states
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  // Handle cancelled state
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // Handle running state
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  // Only analyze issue status for finished, successful executions
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    
    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return "clear";
    }

    // Extract the Prometheus response from the first output payload
    const payload = outputs.default[0];
    if (!payload || !payload.data) {
      return "clear";
    }

    const responseData = payload.data as PrometheusResponse | undefined;
    if (!responseData || !responseData.data || !responseData.data.result) {
      return "clear";
    }

    const results = responseData.data.result;
    
    // No issues found
    if (results.length === 0) {
      return "clear";
    }

    // Analyze issue statuses
    let hasCritical = false;
    let hasDegraded = false;

    for (const result of results) {
      // For instant queries, check the value field: [timestamp, "status"]
      if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
        const status = String(result.value[1]);
        if (status === "2") {
          hasCritical = true;
        } else if (status === "1") {
          hasDegraded = true;
        }
      }
    }

    // Return critical if there's at least one critical issue
    if (hasCritical) {
      return "critical";
    }

    // Return degraded if there are only degraded issues
    if (hasDegraded) {
      return "degraded";
    }

    // Default to clear if we can't determine status
    return "clear";
  }

  return "failed";
};

export const LIST_ISSUES_STATE_REGISTRY: EventStateRegistry = {
  stateMap: LIST_ISSUES_STATE_MAP,
  getState: listIssuesStateFunction,
};

function getIssueCounts(execution: WorkflowsWorkflowNodeExecution): { critical: number; degraded: number } {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  
  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return { critical: 0, degraded: 0 };
  }

  const payload = outputs.default[0];
  if (!payload || !payload.data) {
    return { critical: 0, degraded: 0 };
  }

  const responseData = payload.data as PrometheusResponse | undefined;
  if (!responseData || !responseData.data || !responseData.data.result) {
    return { critical: 0, degraded: 0 };
  }

  const results = responseData.data.result;
  
  let critical = 0;
  let degraded = 0;

  for (const result of results) {
    if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
      const status = String(result.value[1]);
      if (status === "2") {
        critical++;
      } else if (status === "1") {
        degraded++;
      }
    }
  }

  return { critical, degraded };
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const { critical, degraded } = getIssueCounts(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  // Build subtitle with counts and time
  const countParts: string[] = [];
  if (critical > 0) {
    countParts.push(`${critical} critical`);
  }
  if (degraded > 0) {
    countParts.push(`${degraded} degraded`);
  }

  let eventSubtitle: string;
  if (countParts.length > 0) {
    eventSubtitle = `${countParts.join(", ")} · ${timeAgo}`;
  } else {
    eventSubtitle = timeAgo;
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id,
    },
  ];
}
