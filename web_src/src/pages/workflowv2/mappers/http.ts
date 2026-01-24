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
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from ".";

// Custom state map for HTTP component with error state
const HTTP_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-violet-100 dark:bg-violet-900/50",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
  error: {
    icon: "alert-triangle",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-gray-50 dark:bg-gray-700",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-orange-100 dark:bg-orange-900/50",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-black dark:text-gray-200",
    backgroundColor: "bg-sky-100 dark:bg-sky-900/50",
    badgeColor: "bg-blue-500",
  },
};

// Custom state function for HTTP component
const httpStateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED") {
    const metadata = execution.metadata as Record<string, unknown> | undefined;

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
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _items?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      iconSlug: componentDefinition.icon || "globe",
      iconColor: getColorClass("black"),
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0] ? getHTTPEventSections(nodes, lastExecutions[0], httpStateFunction) : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getHTTPMetadataList(node),
      specs: getHTTPSpecs(node),
      eventStateMap: HTTP_EVENT_STATE_MAP,
    };
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = execution.metadata as Record<string, unknown> | undefined;

    if (metadata?.finalStatus !== undefined && metadata.finalStatus !== null) {
      details["Response"] = metadata.finalStatus.toString();
    } else {
      const outputs = execution.outputs as Record<string, unknown>;
      const defaultArray = outputs?.default as unknown[];
      const response = defaultArray?.[0] as {
        data?: {
          status?: number;
        };
      };
      if (response?.data?.status) {
        details["Response"] = response.data.status.toString();
      }
    }

    if (metadata?.totalRetries !== undefined && metadata.totalRetries !== null) {
      details["Retries"] = metadata.totalRetries.toString();
    }

    if (execution.createdAt && execution.updatedAt) {
      const startTime = new Date(execution.createdAt);
      const endTime = new Date(execution.updatedAt);
      const durationMs = endTime.getTime() - startTime.getTime();

      if (durationMs < 1000) {
        details["Duration"] = `${durationMs}ms`;
      } else if (durationMs < 60000) {
        details["Duration"] = `${Math.round(durationMs / 1000)} seconds`;
      } else {
        const minutes = Math.floor(durationMs / 60000);
        const seconds = Math.round((durationMs % 60000) / 1000);
        details["Duration"] = seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
      }
    }

    // Time finished
    if (execution.updatedAt) {
      const finishedTime = new Date(execution.updatedAt);
      details["Time Finished"] = finishedTime.toLocaleDateString("en-US", {
        year: "numeric",
        month: "long",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
        second: "2-digit",
        hour12: true,
      });
    }

    return details;
  },

  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    const state = httpStateFunction(execution);

    // For running state, show duration
    if (state === "running") {
      if (execution.createdAt) {
        const startTime = new Date(execution.createdAt);
        const now = new Date();
        const durationMs = now.getTime() - startTime.getTime();

        if (durationMs < 60000) {
          return `Running for: ${Math.floor(durationMs / 1000)}s`;
        } else {
          const minutes = Math.floor(durationMs / 60000);
          return `Running for: ${minutes}m`;
        }
      }
      return "Running...";
    }

    // For success/failed states, show response code and time
    if (state === "success" || state === "failed") {
      const metadata = execution.metadata as Record<string, unknown> | undefined;
      let responseCode: string | null = null;

      // Try to get response code from metadata first
      if (metadata?.finalStatus !== undefined && metadata.finalStatus !== null) {
        responseCode = metadata.finalStatus.toString();
      } else {
        // Fallback to outputs
        const outputs = execution.outputs as Record<string, unknown>;
        const defaultArray = outputs?.default as unknown[];
        const response = defaultArray?.[0] as {
          data?: { status?: number };
        };
        if (response?.data?.status) {
          responseCode = response.data.status.toString();
        }
      }

      const timeAgo = execution.updatedAt ? formatTimeAgo(new Date(execution.updatedAt)) : "";

      if (responseCode && timeAgo) {
        return `Response: ${responseCode} · ${timeAgo}`;
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (timeAgo) {
        return timeAgo;
      }
    }

    // Fallback: just show time ago
    if (execution.updatedAt) {
      return formatTimeAgo(new Date(execution.updatedAt));
    }

    return "";
  },
};

function getHTTPMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: Array<{ icon: string; label: string }> = [];

  // Method and URL
  if (node.configuration?.url && node.configuration.method) {
    metadata.push({
      icon: "link",
      label: `${node.configuration.method} ${node.configuration.url}`,
    });
  }

  // Request body information
  const contentType = node.configuration?.contentType;
  if (
    contentType &&
    node.configuration &&
    (node.configuration.method === "POST" ||
      node.configuration.method === "PUT" ||
      node.configuration.method === "PATCH")
  ) {
    let bodyLabel = "";

    switch (contentType) {
      case "application/json":
        bodyLabel = "JSON body";
        break;
      case "application/xml":
        bodyLabel = "XML body";
        break;
      case "text/plain":
        bodyLabel = "Text body";
        break;
      case "application/x-www-form-urlencoded":
        bodyLabel = "Form data";
        break;
      default:
        bodyLabel = "Request body";
    }

    metadata.push({
      icon: "code",
      label: bodyLabel,
    });
  }

  // Headers count
  const headers = node.configuration?.headers as Array<{ name: string; value: string }> | undefined;
  if (headers && headers.length > 0) {
    metadata.push({
      icon: "code",
      label: `${headers.length} header${headers.length === 1 ? "" : "s"}`,
    });
  }

  // Retry configuration
  const timeoutStrategy = node.configuration?.timeoutStrategy;
  const retries = node.configuration?.retries as number | undefined;
  const timeoutSeconds = node.configuration?.timeoutSeconds as number | undefined;

  if (timeoutStrategy && retries !== undefined && timeoutSeconds !== undefined) {
    const retriesText = retries === 0 ? "No retries" : `${retries} ${retries === 1 ? "retry" : "retries"}`;
    metadata.push({
      icon: "bolt",
      label: `${retriesText}, with ${timeoutSeconds}s timeout`,
    });
  } else if (retries !== undefined && retries > 0) {
    metadata.push({
      icon: "bolt",
      label: `${retries} ${retries === 1 ? "retry" : "retries"}`,
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
    let payload: unknown = null;
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
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  stateFunction: (execution: WorkflowsWorkflowNodeExecution) => EventState,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const generateEventSubtitle = (): string => {
    const state = stateFunction(execution);

    if (state === "running") {
      if (execution.createdAt) {
        const startTime = new Date(execution.createdAt);
        const now = new Date();
        const durationMs = now.getTime() - startTime.getTime();

        if (durationMs < 60000) {
          return `Running for: ${Math.floor(durationMs / 1000)}s`;
        } else {
          const minutes = Math.floor(durationMs / 60000);
          return `Running for: ${minutes}m`;
        }
      }
      return "Running...";
    }

    if (state === "success" || state === "failed") {
      const metadata = execution.metadata as Record<string, unknown> | undefined;
      let responseCode: string | null = null;

      if (metadata?.finalStatus !== undefined && metadata.finalStatus !== null) {
        responseCode = metadata.finalStatus.toString();
      } else {
        const outputs = execution.outputs as Record<string, unknown>;
        const defaultArray = outputs?.default as unknown[];
        const response = defaultArray?.[0] as {
          data?: { status?: number };
        };
        if (response?.data?.status) {
          responseCode = response.data.status.toString();
        }
      }

      const timeAgo = execution.updatedAt ? formatTimeAgo(new Date(execution.updatedAt)) : "";

      if (responseCode && timeAgo) {
        return `Response: ${responseCode} · ${timeAgo}`;
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (timeAgo) {
        return timeAgo;
      }
    }

    // Fallback: just show time ago
    if (execution.updatedAt) {
      return formatTimeAgo(new Date(execution.updatedAt));
    }

    return "";
  };

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle: generateEventSubtitle(),
    eventState: stateFunction(execution),
    eventId: execution.rootEvent?.id,
    showAutomaticTime: stateFunction(execution) === "running", // Show live timer for running state
  };

  return [eventSection];
}
