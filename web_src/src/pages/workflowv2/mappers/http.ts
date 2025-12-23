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

  const sendBody = node.configuration?.sendBody;
  const contentType = node.configuration?.contentType || "application/json";

  // Show payload based on content type if sendBody is enabled
  if (sendBody) {
    let payload: any = null;
    let payloadIcon = "file-json";
    let payloadTitle = "payload";
    let tooltipContentType: "json" | "xml" | "text" = "json";

    switch (contentType) {
      case "application/json":
        payload = node.configuration?.payload;
        payloadIcon = "file-json";
        payloadTitle = "json payload";
        tooltipContentType = "json";
        break;
      case "application/x-www-form-urlencoded":
        payload = node.configuration?.payloadFormData;
        payloadIcon = "list";
        payloadTitle = "form data";
        tooltipContentType = "json"; // Form data is shown as badges, not tooltip
        break;
      case "text/plain":
        payload = node.configuration?.payloadText;
        payloadIcon = "file-text";
        payloadTitle = "text payload";
        tooltipContentType = "text";
        break;
      case "application/xml":
        payload = node.configuration?.payloadXML;
        payloadIcon = "file-code";
        payloadTitle = "xml payload";
        tooltipContentType = "xml";
        break;
    }

    // Only show payload spec if there's actual content
    if (payload) {
      const hasContent =
        (typeof payload === "object" && !Array.isArray(payload) && Object.keys(payload).length > 0) ||
        (typeof payload === "string" && payload.length > 0) ||
        (Array.isArray(payload) && payload.length > 0);

      if (hasContent) {
        // For form data, show as badges like headers
        if (contentType === "application/x-www-form-urlencoded" && Array.isArray(payload)) {
          specs.push({
            title: payloadTitle,
            tooltipTitle: "form data parameters",
            iconSlug: payloadIcon,
            values: payload.map((param: { key: string; value: string }) => ({
              badges: [
                {
                  label: param.key,
                  bgColor: "bg-green-100",
                  textColor: "text-green-800",
                },
                {
                  label: param.value,
                  bgColor: "bg-gray-100",
                  textColor: "text-gray-800",
                },
              ],
            })),
          });
        } else {
          specs.push({
            title: payloadTitle,
            tooltipTitle: `request ${payloadTitle}`,
            iconSlug: payloadIcon,
            value: payload,
            contentType: tooltipContentType,
          });
        }
      }
    }
  }

  const sendHeaders = node.configuration?.sendHeaders;
  const headers = node.configuration?.headers as Array<{ name: string; value: string }> | undefined;
  if (sendHeaders && headers && headers.length > 0) {
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
