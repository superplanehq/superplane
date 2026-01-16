import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { ListIssuesConfiguration } from "./types";

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

    return {
      iconSrc: dash0Icon,
      iconBackground: "bg-white",
      headerColor: "bg-white",
      collapsedBackground: "bg-white",
      collapsed: node.isCollapsed,
      title: node.name!,
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
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

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as unknown as ListIssuesConfiguration;

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: `Dataset: ${configuration.dataset}` });
  }

  if (configuration?.checkRules && configuration.checkRules.length > 0) {
    const checkRulesLabel =
      configuration.checkRules.length === 1
        ? `Check Rule: ${configuration.checkRules[0]}`
        : `Check Rules: ${configuration.checkRules.length} selected`;
    metadata.push({ icon: "check-circle", label: checkRulesLabel });
  }

  metadata.push({ icon: "alert-triangle", label: "Query: {otel_metric_name=\"dash0.issue.status\"} >= 1" });

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
