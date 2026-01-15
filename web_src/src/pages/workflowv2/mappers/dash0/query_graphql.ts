import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { QueryGraphQLConfiguration } from "./types";

export const queryGraphQLMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    return {
      iconSrc: dash0Icon,
      iconBackground: "bg-blue-500",
      headerColor: getBackgroundColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name!,
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default: OutputPayload[] };
    const responseData = outputs.default[0]?.data as Record<string, any>;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    // Format the GraphQL response data for display
    const details: Record<string, string> = {};
    try {
      const formatted = JSON.stringify(responseData, null, 2);
      details["Response Data"] = formatted;
    } catch (error) {
      details["Response Data"] = String(responseData);
    }

    return details;
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryGraphQLConfiguration;

  if (configuration?.query) {
    // Show a preview of the query (first 50 chars)
    const queryPreview = configuration.query.length > 50
      ? configuration.query.substring(0, 50) + "..."
      : configuration.query;
    metadata.push({ icon: "code", label: queryPreview });
  }

  if (configuration?.variables && Object.keys(configuration.variables).length > 0) {
    metadata.push({
      icon: "settings",
      label: `${Object.keys(configuration.variables).length} variable(s)`,
    });
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
