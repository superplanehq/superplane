import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { MetadataItem } from "@/ui/metadataList";
import { OutputPayload, ComponentBaseMapper } from "../types";
import { Issue } from "./types";

export const baseIssueMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    queueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default: OutputPayload[] };
    const issue = outputs.default[0].data as Issue;
    return getDetailsForIssue(issue);
  },
};

export function baseProps(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  componentDefinition: ComponentsComponent,
  lastExecutions: WorkflowsWorkflowNodeExecution[],
  _?: WorkflowsWorkflowNodeQueueItem[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name!;

  return {
    iconSrc: githubIcon,
    iconColor: getColorClass(componentDefinition.color),
    headerColor: getBackgroundColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name!,
    eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: metadataList(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

export function getDetailsForIssue(issue: Issue): Record<string, string> {
  const details: Record<string, string> = {
    Number: issue?.number.toString(),
    ID: issue?.id.toString(),
    State: issue?.state,
    URL: issue?.html_url,
    Title: issue?.title || "-",
    Author: issue?.user?.html_url || "-",
    "Created At": issue?.created_at,
  };

  if (issue.closed_by) {
    details["Closed By"] = issue?.closed_by.html_url;
    details["Closed At"] = issue?.closed_at!;
  }

  if (issue.labels) {
    details["Labels"] = issue.labels.map((label) => label.name).join(", ");
  }

  if (issue.assignees) {
    details["Assignees"] = issue.assignees.map((assignee) => assignee.login).join(", ");
  }

  return details;
}

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  return metadata;
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
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
      eventId: execution.rootEvent?.id,
    },
  ];
}
