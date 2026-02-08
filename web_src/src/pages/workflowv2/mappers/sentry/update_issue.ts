import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";

interface UpdateIssueConfiguration {
  organization?: string;
  issueId?: string;
  status?: string;
  assignedTo?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;

  if (configuration?.issueId) {
    metadata.push({ icon: "hash", label: configuration.issueId });
  }
  if (configuration?.status) {
    metadata.push({ icon: "circle-dot", label: `Status: ${configuration.status}` });
  }
  if (configuration?.assignedTo) {
    metadata.push({ icon: "user", label: configuration.assignedTo });
  }

  return metadata;
}

function updateIssueEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? execution.id,
    },
  ];
}

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "sentry.updateIssue";

    return {
      title:
        context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Update Issue",
      iconSrc: sentryIcon,
      iconSlug: "alert-triangle",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? updateIssueEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: metadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    const updatedAt = context.execution.updatedAt || context.execution.createdAt;

    return {
      "Updated At": updatedAt ? formatTimeAgo(new Date(updatedAt)) : "-",
      "Issue ID": stringOrDash(result?.id),
      "Short ID": stringOrDash(result?.shortId),
      Status: stringOrDash(result?.status),
      Title: stringOrDash(result?.title),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};
