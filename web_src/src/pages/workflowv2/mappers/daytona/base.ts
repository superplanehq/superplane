import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import { formatTimeAgo } from "@/utils/date";

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: daytonaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, any> | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};
    if (payload?.timestamp) {
      details["Executed At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (context.node.componentName === "daytona.getPreviewUrl") {
      if (typeof responseData.sandbox === "string" && responseData.sandbox.length > 0) {
        details["Sandbox"] = responseData.sandbox;
      }

      if (typeof responseData.port === "number") {
        details["Port"] = String(responseData.port);
      }

      if (typeof responseData.signed === "boolean") {
        details["Signed URL"] = responseData.signed ? "true" : "false";
      }

      if (typeof responseData.expiresInSeconds === "number") {
        details["Expires In Seconds"] = String(responseData.expiresInSeconds);
      }

      if (typeof responseData.token === "string" && responseData.token.length > 0) {
        details["Token"] = responseData.token;
      }

      if (typeof responseData.url === "string" && responseData.url.length > 0) {
        details["Preview URL"] = responseData.url;
      }
    }

    try {
      const formatted = JSON.stringify(responseData, null, 2);
      details["Response"] = formatted;
    } catch (error) {
      details["Response"] = String(responseData);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
