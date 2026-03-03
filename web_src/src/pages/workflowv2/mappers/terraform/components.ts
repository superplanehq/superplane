import {
  ComponentBaseProps,
  EventSection,
  DEFAULT_EVENT_STATE_MAP,
  EventStateMap,
  EventState,
} from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getTriggerRenderer } from "..";
import terraformIcon from "@/assets/icons/integrations/terraform.svg";
import { MetadataItem } from "@/ui/metadataList";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  NodeInfo,
  ExecutionInfo,
  StateFunction,
  EventStateRegistry,
} from "../types";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { formatTimeAgo } from "@/utils/date";

export const TERRAFORM_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  needsAttention: {
    icon: "alert-circle",
    textColor: "text-orange-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-orange-500",
  },
  passed: {
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
};

export const terraformStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";
  if (execution.result === "RESULT_FAILED" || execution.resultReason === "RESULT_REASON_ERROR") return "failed";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";

  const metadata = execution.metadata as Record<string, any>;
  const currentStatus = metadata?.currentStatus;

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    const needsAttentionStates = [
      "planned",
      "cost_estimated",
      "policy_checked",
      "policy_override",
      "planned_and_saved",
    ];
    if (needsAttentionStates.includes(currentStatus)) {
      return "needsAttention";
    }
    return "running";
  }

  const isFailedState = ["discarded", "errored", "canceled", "policy_soft_failed", "force_canceled"].includes(
    currentStatus,
  );
  if (isFailedState) return "failed";

  return "passed";
};

export const TERRAFORM_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TERRAFORM_STATE_MAP,
  getState: terraformStateFunction,
};

export const terraformComponentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    const metadata: MetadataItem[] = [];
    const config = node.configuration as Record<string, any>;
    const nodeMetadata = node.metadata as Record<string, any>;
    if (nodeMetadata?.workspace?.name) {
      metadata.push({ icon: "box", label: nodeMetadata.workspace.name });
    } else if (config?.workspaceId) {
      metadata.push({ icon: "box", label: config.workspaceId });
    }

    const executionMetadata = lastExecution?.metadata as Record<string, any>;
    if (executionMetadata?.runId) {
      metadata.push({ icon: "play", label: executionMetadata.runId });
    }

    return {
      iconSrc: terraformIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution) : undefined,
      metadata,
      includeEmptyState: !lastExecution,
      eventStateMap: TERRAFORM_STATE_MAP,
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    const timeStr = timestamp ? formatTimeAgo(new Date(timestamp)) : "";

    const metadata = context.execution.metadata as Record<string, any>;
    if (metadata) {
      const parts: string[] = [];
      if (metadata.workspaceName) parts.push(metadata.workspaceName);
      if (metadata.runId) parts.push(metadata.runId);

      if (parts.length > 0) {
        return timeStr ? `${parts.join(" - ")} • ${timeStr}` : parts.join(" - ");
      }
    }

    return timeStr;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const metadata = context.execution.metadata as Record<string, any>;

    if (!metadata) return details;

    if (metadata.runId) details["Run ID"] = metadata.runId;
    if (metadata.workspaceName) details["Workspace Name"] = metadata.workspaceName;
    if (metadata.currentStatus) details["Current Status"] = metadata.currentStatus;
    if (metadata.runUrl) details["Run URL"] = metadata.runUrl;

    if (metadata.stateHistory && Array.isArray(metadata.stateHistory) && metadata.stateHistory.length > 0) {
      details["State History"] = {
        __type: "terraformStates",
        states: metadata.stateHistory,
      };
    }

    return details;
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });
  const executionState = terraformStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: executionState,
      eventSubtitle: subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined,
      eventId: execution.rootEvent!.id!,
    },
  ];
}
