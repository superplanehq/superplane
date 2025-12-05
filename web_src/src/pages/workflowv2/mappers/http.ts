import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { success, failed, neutral, running } from "./eventSectionUtils";

export const httpMapper: ComponentBaseMapper = {
  props(
    _: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _items?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      iconSlug: componentDefinition.icon || "globe",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getHTTPEventSections(lastExecutions[0]),
      metadata: getHTTPMetadataList(node),
      specs: getHTTPSpecs(node),
    };
  },
};

function getHTTPMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: Array<{ icon: string; label: string }> = [];

  if (node.configuration?.url && node.configuration.method) {
    metadata.push({
      icon: "link",
      label: `${node.configuration.method} ${node.configuration.url}`,
    });
  }

  return metadata;
}

function getHTTPSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];

  const payload = node.configuration?.payload;
  if (payload && Object.keys(payload).length > 0) {
    specs.push({
      title: "payload",
      tooltipTitle: "request payload",
      iconSlug: "file-json",
      value: payload,
    });
  }

  const headers = node.configuration?.headers as Array<{ name: string; value: string }> | undefined;
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

  return specs;
}

function getHTTPEventSections(execution: WorkflowsWorkflowNodeExecution): EventSection[] {
  if (!execution) {
    return [
      neutral({
        title: "Last Run",
        eventTitle: "No events received yet",
      }),
    ];
  }

  const outputs = execution.outputs as any;
  const response = outputs?.default?.[0];

  const baseProps = {
    title: "Last Run",
    receivedAt: new Date(execution.createdAt!),
    eventTitle: execution.state == "STATE_FINISHED" ? `Status: ${response?.status}` : "Running...",
  };

  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return [running(baseProps)];
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return [success(baseProps)];
  }

  return [failed(baseProps)];
}
