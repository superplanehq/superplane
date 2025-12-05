import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass } from "@/utils/colors";
import { success, failed, neutral, running } from "./eventSectionUtils";

export const noopMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    return {
      iconSlug: componentDefinition.icon || "circle-off",
      headerColor: "bg-gray-50",
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getNoopEventSections(nodes, lastExecution),
    };
  },
};

function getNoopEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution | null,
): EventSection[] {
  if (!execution) {
    return [
      neutral({
        title: "Last Run",
        eventTitle: "No events received yet",
      }),
    ];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const baseProps = {
    title: "Last Run",
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
  };

  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return [running(baseProps)];
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return [success(baseProps)];
  }

  return [failed(baseProps)];
}
