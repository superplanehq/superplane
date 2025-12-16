import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getState, getStateMap } from ".";

export const httpMapper: ComponentBaseMapper = {
  props(
    _: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _items?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "http";

    return {
      iconSlug: componentDefinition.icon || "globe",
      headerColor: "bg-white",
      iconColor: getColorClass("black"),
      iconBackground: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0] ? getHTTPEventSections(lastExecutions[0], componentName) : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getHTTPMetadataList(node),
      specs: getHTTPSpecs(node),
      eventStateMap: getStateMap(componentName),
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

function getHTTPEventSections(execution: WorkflowsWorkflowNodeExecution, componentName: string): EventSection[] {
  const outputs = execution.outputs as Record<string, unknown>;
  const defaultArray = outputs?.default as unknown[];
  const response = defaultArray?.[0] as { status?: string };

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: execution.state == "STATE_FINISHED" ? `Status: ${response?.status}` : "Running...",
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}
