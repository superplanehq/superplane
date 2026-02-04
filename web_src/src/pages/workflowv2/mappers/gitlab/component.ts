import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { MetadataItem } from "@/ui/metadataList";
import { OutputPayload, ComponentBaseMapper } from "../types";
import { Issue, GitLabNodeMetadata } from "./types";

export const createIssueMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    queueItems?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const issue = outputs.default[0].data as Issue;
      return `#${issue.iid} ${issue.title}`;
    }
    return "Issue Created";
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    if (!outputs.default[0].data) {
      return details;
    }

    const issue = outputs.default[0].data as Issue;
    return { ...getDetailsForIssue(issue), ...details };
  },
};

function baseProps(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  componentDefinition: ComponentsComponent,
  lastExecutions: CanvasesCanvasNodeExecution[],
  _?: CanvasesCanvasNodeQueueItem[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.component?.name || "unknown";

  return {
    iconSrc: gitlabIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
    eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: metadataList(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function getDetailsForIssue(issue: Issue): Record<string, string> {
  const details: Record<string, string> = {};
  Object.assign(details, {
    "Created At": issue?.created_at ? new Date(issue.created_at).toLocaleString() : "-",
    "Created By": issue?.author?.username || "-",
  });

  details["IID"] = issue?.iid.toString();
  details["ID"] = issue?.id.toString();
  details["State"] = issue?.state;
  details["URL"] = issue?.web_url;
  details["Title"] = issue?.title || "-";

  if (issue.closed_by) {
    details["Closed By"] = issue?.closed_by.username;
    details["Closed At"] = issue?.closed_at ? new Date(issue.closed_at).toLocaleString() : "";
  }

  if (issue.labels && issue.labels.length > 0) {
    details["Labels"] = issue.labels.join(", ");
  }

  if (issue.assignees && issue.assignees.length > 0) {
    details["Assignees"] = issue.assignees.map((assignee) => assignee.username).join(", ");
  }

  return details;
}

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as unknown as GitLabNodeMetadata;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  return metadata;
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventSubtitle: formatTimeAgo(new Date(execution.updatedAt || execution.createdAt || "")),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
