import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import type { MetadataItem } from "@/ui/metadataList";
import type { NodeInfo, ComponentDefinition, ExecutionInfo } from "../types";
import type { NodeMetadata } from "./types";
import { buildBitbucketExecutionSubtitle } from "./utils";

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
  const nodeMetadata = node.metadata as unknown as NodeMetadata;
  const repository = nodeMetadata?.repository;

  if (repository) {
    metadata.push({ icon: "book", label: repository.full_name || repository.name || "" });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventSubtitle: buildBitbucketExecutionSubtitle(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
