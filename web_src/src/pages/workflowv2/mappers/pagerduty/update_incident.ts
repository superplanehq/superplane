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
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";
import { formatTimeAgo } from "@/utils/date";

export const updateIncidentMapper: ComponentBaseMapper = {
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
      iconSrc: pdIcon,
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
    const incident = outputs.default[0].data.incident as Incident;
    return getDetailsForIncident(incident);
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  // Show which fields are being updated
  const updates: string[] = [];
  if (configuration.status) {
    updates.push(`Status: ${configuration.status}`);
  }
  if (configuration.priority) {
    updates.push("Priority");
  }
  if (configuration.title) {
    updates.push("Title");
  }
  if (configuration.escalationPolicy) {
    updates.push("Escalation Policy");
  }
  if (configuration.assignees && configuration.assignees.length > 0) {
    updates.push(`Assignees (${configuration.assignees.length})`);
  }
  if (configuration.note) {
    updates.push("Note");
  }

  if (updates.length > 0) {
    metadata.push({ icon: "funnel", label: `Updating: ${updates.join(", ")}` });
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
