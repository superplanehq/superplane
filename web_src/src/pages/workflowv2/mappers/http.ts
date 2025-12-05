import { ComponentsComponent, ComponentsNode, WorkflowsWorkflowNodeExecution } from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";

export const httpMapper: ComponentBaseMapper = {
  props(
    _: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecution: WorkflowsWorkflowNodeExecution,
  ): ComponentBaseProps {
    const headers = node.configuration?.headers as Array<{ name: string; value: string }> | undefined;
    const payload = node.configuration?.payload;

    // Build metadata array with method and URL
    const metadata: Array<{ icon: string; label: string }> = [];

    if (node.configuration?.url && node.configuration.method) {
      metadata.push({
        icon: "link",
        label: `${node.configuration.method} ${node.configuration.url}`,
      });
    }

    // Build specs array for headers and payload
    const specs: ComponentBaseSpec[] = [];

    // Add payload spec
    if (payload !== undefined && payload !== null) {
      specs.push({
        title: "payload",
        tooltipTitle: "request payload",
        iconSlug: "file-json",
        value: payload,
      });
    }

    // Add headers spec
    if (headers && headers.length > 0) {
      specs.push({
        title: "header",
        tooltipTitle: "request headers",
        iconSlug: "list",
        values: headers.map((header) => ({
          badges: [
            {
              label: header.name,
              bgColor: "bg-blue-100",
              textColor: "text-blue-800",
            },
            {
              label: header.value,
              bgColor: "bg-gray-100",
              textColor: "text-gray-800",
            },
          ],
        })),
      });
    }

    return {
      iconSlug: componentDefinition.icon || "globe",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getHTTPEventSections(lastExecution),
      metadata,
      specs: specs.length > 0 ? specs : undefined,
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
