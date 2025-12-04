import { ComponentsComponent, ComponentsNode, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";

export const httpMapper: ComponentBaseMapper = {
  props(
    _: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecution: WorkflowsWorkflowNodeExecution,
  ): ComponentBaseProps {
    return {
      iconSlug: componentDefinition.icon || "globe",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getHTTPEventSections(lastExecution),
      metadata: [
        {
          icon: "link",
          label: node.configuration?.url as string,
        },
        {
          icon: "globe",
          label: node.configuration?.method as string,
        }
      ]
    };
  },
};

function getHTTPEventSections(execution: WorkflowsWorkflowNodeExecution): EventSection[] {
  if (!execution) {
    return [
      {
        title: "Last Run",
        eventTitle: "No events received yet",
        eventState: "neutral" as const,
      },
    ];
  }

  const outputs = execution.outputs as any;
  const response = outputs?.default?.[0];

  return [
    {
      title: "Last Run",
      receivedAt: new Date(execution.createdAt!),
      eventTitle: execution.state == "STATE_FINISHED" ? `Status: ${response?.status}` : "Running...",
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
