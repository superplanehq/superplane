import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import type {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventStateMap,
  EventState,
} from "@/ui/componentBase";
import type React from "react";
import { getColorClass } from "@/utils/colors";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import { getTriggerRenderer } from ".";
import { stringOrDash } from "./utils";

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
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-black",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
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
const httpStateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
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
    const outputs = execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] } | undefined;
    if (outputs?.success) {
      return "success";
    } else if (outputs?.failure) {
      return "failed";
    }
  }

  return "failed";
};

export const HTTP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: HTTP_EVENT_STATE_MAP,
  getState: httpStateFunction,
};

interface Output {
  status: number;
}

interface Metadata {
  retry: {
    strategy: string;
    interval: number;
    nextRetryAt: string;
    attempts: number;
    maxAttempts: number;
    lastStatus: number | null;
    lastResponse: string | null;
    lastError: string | null;
  };
}

export const httpMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      iconSlug: context.componentDefinition.icon || "globe",
      iconColor: getColorClass("black"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: context.lastExecutions[0]
        ? getHTTPEventSections(context.nodes, context.lastExecutions[0], httpStateFunction)
        : undefined,
      includeEmptyState: !context.lastExecutions[0],
      metadata: getHTTPMetadataList(context.node),
      specs: getHTTPSpecs(context.node),
      eventStateMap: HTTP_EVENT_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = context.execution.metadata as Metadata | undefined;
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] };

    if (outputs?.success) {
      const response = outputs.success[0].data as Output;
      details["Response"] = response.status.toString();
    } else if (outputs?.failure) {
      const response = outputs.failure[0].data as Output;
      details["Response"] = response.status.toString();
    }

    if (metadata?.retry) {
      details["Retries"] = `${metadata.retry.attempts} / ${metadata.retry.maxAttempts}`;
      details["Retry Strategy"] = metadata.retry.strategy;
      details["Next Retry At"] = metadata.retry.nextRetryAt;
      details["Last Status"] = stringOrDash(metadata.retry.lastStatus);
      details["Last Response"] = stringOrDash(metadata.retry.lastResponse);
      details["Last Error"] = stringOrDash(metadata.retry.lastError);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const state = httpStateFunction(context.execution);

    // For running state, show duration
    if (state === "running") {
      if (context.execution.createdAt) {
        const startTime = new Date(context.execution.createdAt);
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
      const outputs = context.execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] };
      let responseCode: string | null = null;
      if (outputs?.success) {
        const response = outputs.success[0].data as Output;
        responseCode = response.status.toString();
      } else if (outputs?.failure) {
        const response = outputs.failure[0].data as Output;
        responseCode = response.status.toString();
      }

      if (responseCode && context.execution.updatedAt) {
        return renderWithTimeAgo(`Response: ${responseCode}`, new Date(context.execution.updatedAt));
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (context.execution.updatedAt) {
        return renderTimeAgo(new Date(context.execution.updatedAt));
      }
    }

    // Fallback: just show time ago
    if (context.execution.updatedAt) {
      return renderTimeAgo(new Date(context.execution.updatedAt));
    }

    return "";
  },
};

function getHTTPMetadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as HTTPConfiguration;
  const metadata: Array<{ icon: string; label: string }> = [];

  // Method and URL
  if (configuration.url && configuration.method) {
    metadata.push({
      icon: "link",
      label: `${configuration.method} ${configuration.url}`,
    });
  }

  // Request body information
  const contentType = configuration.contentType;
  if (
    contentType &&
    node.configuration &&
    (configuration.method === "POST" || configuration.method === "PUT" || configuration.method === "PATCH")
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

  // Retry configuration
  if (configuration.retry && configuration.retry.enabled) {
    const strategy = configuration.retry.strategy;
    const retries = configuration.retry.maxAttempts;
    const interval = configuration.retry.intervalSeconds;
    metadata.push({
      icon: "bolt",
      label: `${retries} retries, ${strategy}, ${interval}s`,
    });
  }

  return metadata;
}

type HTTPConfiguration = {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  contentType: "application/json" | "application/xml" | "text/plain" | "application/x-www-form-urlencoded";
  headers: Array<{ name: string; value: string }>;
  queryParams: Array<{ key: string; value: string }>;
  json?: any;
  formData?: Array<{ key: string; value: string }>;
  text?: string;
  xml?: string;
  timeoutSeconds: number;
  retry?: RetrySpec;
};

type RetrySpec = {
  enabled: boolean;
  strategy: "fixed" | "exponential";
  maxAttempts: number;
  intervalSeconds: number;
};

function getHTTPSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as HTTPConfiguration;

  const contentType = configuration.contentType || "application/json";

  // Show payload based on content type if sendBody is enabled
  let payload: unknown = null;
  let payloadIcon = "file-json";
  let payloadTitle = "payload";
  let tooltipContentType: "json" | "xml" | "text" = "json";

  switch (contentType) {
    case "application/json":
      payload = configuration.json;
      payloadIcon = "file-json";
      payloadTitle = "json payload";
      tooltipContentType = "json";
      break;
    case "application/x-www-form-urlencoded":
      payload = configuration.formData;
      payloadIcon = "list";
      payloadTitle = "form data";
      tooltipContentType = "json"; // Form data is shown as badges, not tooltip
      break;
    case "text/plain":
      payload = configuration.text;
      payloadIcon = "file-text";
      payloadTitle = "text payload";
      tooltipContentType = "text";
      break;
    case "application/xml":
      payload = configuration.xml;
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

  if (configuration.headers && configuration.headers.length > 0) {
    specs.push({
      title: "header",
      tooltipTitle: "request headers",
      iconSlug: "list",
      values: configuration.headers.map((header) => ({
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

  if (configuration.queryParams && configuration.queryParams.length > 0) {
    specs.push({
      title: "query param",
      tooltipTitle: "query params",
      iconSlug: "list",
      values: configuration.queryParams.map((param) => ({
        badges: [
          {
            label: param.key,
            bgColor: "bg-blue-100",
            textColor: "text-blue-800",
          },
          {
            label: param.value,
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
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  stateFunction: (execution: ExecutionInfo) => EventState,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const generateEventSubtitle = (): string | React.ReactNode => {
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
        const outputs = execution.outputs as { success?: OutputPayload[]; failure?: OutputPayload[] };
        if (outputs?.success) {
          const response = outputs.success[0].data as Output;
          responseCode = response.status.toString();
        } else if (outputs?.failure) {
          const response = outputs.failure[0].data as Output;
          responseCode = response.status.toString();
        }
      }

      if (responseCode && execution.updatedAt) {
        return renderWithTimeAgo(`Response: ${responseCode}`, new Date(execution.updatedAt));
      } else if (responseCode) {
        return `Response: ${responseCode}`;
      } else if (execution.updatedAt) {
        return renderTimeAgo(new Date(execution.updatedAt));
      }
    }

    // Fallback: just show time ago
    if (execution.updatedAt) {
      return renderTimeAgo(new Date(execution.updatedAt));
    }

    return "";
  };

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle: generateEventSubtitle(),
    eventState: stateFunction(execution),
    eventId: execution.rootEvent!.id!,
    showAutomaticTime: stateFunction(execution) === "running", // Show live timer for running state
  };

  return [eventSection];
}
