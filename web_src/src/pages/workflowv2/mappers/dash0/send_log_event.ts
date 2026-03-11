import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { SendLogEventConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const sendLogEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Status: "No response data" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, unknown> | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Sent At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (responseData?.sent) {
      details["Status"] = "Successfully sent";
    }

    if (responseData?.severityText) {
      details["Severity"] = String(responseData.severityText);
    }

    if (responseData?.body) {
      const bodyText = String(responseData.body);
      details["Body"] = bodyText.length > 100 ? bodyText.substring(0, 100) + "..." : bodyText;
    }

    if (responseData?.eventName) {
      details["Event Name"] = String(responseData.eventName);
    }

    if (responseData?.serviceName) {
      details["Service Name"] = String(responseData.serviceName);
    }

    if (responseData?.dataset) {
      details["Dataset"] = String(responseData.dataset);
    }

    if (responseData?.attributes && typeof responseData.attributes === "object") {
      const attrs = responseData.attributes as Record<string, unknown>;
      const attrCount = Object.keys(attrs).length;
      if (attrCount > 0) {
        details["Attributes"] = `${attrCount} attribute${attrCount > 1 ? "s" : ""}`;
      }
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as SendLogEventConfiguration;

  if (configuration?.body) {
    // Show a preview of the log body (first 50 chars)
    const bodyPreview =
      configuration.body.length > 50 ? configuration.body.substring(0, 50) + "..." : configuration.body;
    metadata.push({ icon: "file-text", label: bodyPreview });
  }

  if (configuration?.severityText) {
    metadata.push({ icon: "alert-circle", label: `Severity: ${configuration.severityText}` });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: `Dataset: ${configuration.dataset}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.createdAt || !execution.rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const triggerComponentName = rootTriggerNode?.componentName ?? "";
  const rootTriggerRenderer = getTriggerRenderer(triggerComponentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
