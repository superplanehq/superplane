import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import SentryLogo from "@/assets/icons/integrations/sentry.svg";
import { formatTimeAgo } from "@/utils/date";
import { buildActionStateRegistry } from "../utils";

interface UpdateIssueConfiguration {
  issueId?: string;
  status?: string;
  assignedTo?: string;
  hasSeen?: boolean;
  isBookmarked?: boolean;
}

/**
 * Mapper for the "sentry.updateIssue" component type
 */
export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "sentry.updateIssue";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: SentryLogo,
      iconSlug: context.componentDefinition.icon || "sentry",
      iconColor: getColorClass(context.componentDefinition?.color || "purple"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      includeEmptyState: !lastExecution,
      metadata: getUpdateIssueMetadataList(context.node),
      specs: getUpdateIssueSpecs(context.node),
      eventSections: lastExecution ? getUpdateIssueEventSections(context.nodes, lastExecution, componentName) : undefined,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = context.execution.outputs as { default?: { data?: any }[] } | undefined;
    const payload = outputs?.default?.[0]?.data as Record<string, any> | undefined;

    // Add timestamp
    if (context.execution.createdAt) {
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (context.execution.updatedAt) {
      details["Finished At"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    if (!payload || typeof payload !== "object") {
      return details;
    }

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Issue ID", payload.id);
    addDetail("Short ID", payload.shortId);
    addDetail("Status", payload.status);
    addDetail("Issue URL", payload.permalink);

    if (payload.assignedTo) {
      addDetail("Assigned To", payload.assignedTo.name || payload.assignedTo.email);
    }

    return details;
  },
};

function getUpdateIssueMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;

  if (configuration?.issueId) {
    metadata.push({ icon: "hash", label: configuration.issueId });
  }

  if (configuration?.status) {
    metadata.push({ icon: "check-circle", label: configuration.status });
  }

  return metadata;
}

function getUpdateIssueSpecs(_node: NodeInfo): ComponentBaseSpec[] {
  return [];
}

function getUpdateIssueEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] | undefined {
  // Return undefined if no root event
  if (!execution.rootEvent || !execution.rootEvent.id) {
    return undefined;
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({
    event: execution.rootEvent,
  });

  // Get state using the component-specific state function
  const executionState = getState(componentName)(execution);

  // Use updatedAt for subtitle when execution is complete, createdAt when running
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "",
      eventState: executionState,
      eventId: execution.rootEvent.id,
    },
  ];
}

export const UPDATE_ISSUE_STATE_REGISTRY = buildActionStateRegistry("updated");
