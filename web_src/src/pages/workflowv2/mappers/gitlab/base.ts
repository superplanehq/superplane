import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { MetadataItem } from "@/ui/metadataList";
import { NodeInfo, ComponentDefinition, ExecutionInfo } from "../types";
import { GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

export function baseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: ComponentDefinition,
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

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

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GitLabNodeMetadata;

  if (nodeMetadata?.project?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.project.name });
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
      eventSubtitle: buildGitlabExecutionSubtitle(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
