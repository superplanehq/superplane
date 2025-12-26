import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventStateMap, EventState } from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";

// Custom state map for HTTP component with error state
const HTTP_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-violet-100",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-black",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  error: {
    icon: "alert-triangle",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-gray-50",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-black",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-black",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
};

// Custom state function for HTTP component
const httpStateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  // Check metadata result for success/failed/error distinction
  if (execution.state === "STATE_FINISHED") {
    const metadata = execution.metadata as Record<string, any> | undefined;

    if (metadata?.result === "success") {
      return "success";
    }

    if (metadata?.result === "failed") {
      return "failed";
    }

    if (metadata?.result === "error") {
      return "error";
    }

    // Fallback to execution result
    if (execution.result === "RESULT_PASSED") {
      return "success";
    }
  }

  return "failed";
};

export const HTTP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: HTTP_EVENT_STATE_MAP,
  getState: httpStateFunction,
};

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
      headerColor: "bg-white",
      iconColor: getColorClass("black"),
      iconBackground: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0] ? getHTTPEventSections(lastExecutions[0], httpStateFunction) : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getHTTPMetadataList(node),
      specs: getHTTPSpecs(node),
      eventStateMap: HTTP_EVENT_STATE_MAP,
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
        payload = node.configuration?.json;
        payloadIcon = "file-json";
        payloadTitle = "json payload";
        tooltipContentType = "json";
        break;
      case "application/x-www-form-urlencoded":
        payload = node.configuration?.formData;
        payloadIcon = "list";
        payloadTitle = "form data";
        tooltipContentType = "json"; // Form data is shown as badges, not tooltip
        break;
      case "text/plain":
        payload = node.configuration?.text;
        payloadIcon = "file-text";
        payloadTitle = "text payload";
        tooltipContentType = "text";
        break;
      case "application/xml":
        payload = node.configuration?.xml;
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

function getHTTPEventSections(
  execution: WorkflowsWorkflowNodeExecution,
  stateFunction: (execution: WorkflowsWorkflowNodeExecution) => EventState,
): EventSection[] {
  const outputs = execution.outputs as Record<string, unknown>;
  const defaultArray = outputs?.default as unknown[];
  const response = defaultArray?.[0] as { data?: { status?: string; error?: string } };

  // Determine event title based on response
  let eventTitle = "Running...";
  if (execution.state === "STATE_FINISHED") {
    if (response?.data?.error) {
      eventTitle = `Error: ${response.data.error}`;
    } else if (response?.data?.status) {
      eventTitle = `Status: ${response.data.status}`;
    } else {
      eventTitle = "Request completed";
    }
  }

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle,
    eventState: stateFunction(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}
