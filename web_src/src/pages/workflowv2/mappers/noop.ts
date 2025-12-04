import { ComponentsNode, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass } from "@/utils/colors";

export const noopMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    lastExecution: WorkflowsWorkflowNodeExecution,
  ): ComponentBaseProps {
    return {
      iconSlug: "circle-off",
      headerColor: "bg-gray-50",
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getEventSections(nodes, lastExecution),
    };
  },
};

function getEventSections(nodes: ComponentsNode[], execution: WorkflowsWorkflowNodeExecution): EventSection[] {
  if (!execution) {
    return [
      {
        title: "Last Run",
        eventTitle: "No events received yet",
        eventState: "neutral" as const,
      },
    ];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      title: "Last Run",
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: executionToEventSectionState(execution),
    },
  ];
}

function executionToEventSectionState(execution: WorkflowsWorkflowNodeExecution): EventState {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}
