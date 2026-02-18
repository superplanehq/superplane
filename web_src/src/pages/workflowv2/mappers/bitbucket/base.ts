import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import { MetadataItem } from "@/ui/metadataList";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  NodeInfo,
  ComponentDefinition,
  ExecutionInfo,
  OutputPayload,
} from "../types";
import { buildExecutionSubtitle, stringOrDash } from "../utils";
import { Issue, NodeMetadata } from "./types";

export const baseIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    return buildExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const issue = outputs.default[0].data as Issue;
    details["ID"] = stringOrDash(issue?.id);
    details["Title"] = stringOrDash(issue?.title);
    details["State"] = stringOrDash(issue?.state);
    details["URL"] = stringOrDash(issue?.links?.html?.href);
    details["Reporter"] = stringOrDash(issue?.reporter?.display_name);
    details["Assignee"] = stringOrDash(issue?.assignee?.display_name);
    details["Created At"] = issue?.created_on ? new Date(issue.created_on).toLocaleString() : "-";
    details["Updated At"] = issue?.updated_on ? new Date(issue.updated_on).toLocaleString() : "-";

    return details;
  },
};

export function baseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: ComponentDefinition,
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

  return {
    iconSrc: bitbucketIcon,
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

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as NodeMetadata;

  if (nodeMetadata?.repository) {
    metadata.push({
      icon: "book",
      label: nodeMetadata.repository.full_name || nodeMetadata.repository.name || "-",
    });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventSubtitle: buildExecutionSubtitle(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
