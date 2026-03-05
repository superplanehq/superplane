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
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import { formatTimeAgo } from "@/utils/date";

interface PublishMessageConfiguration {
  topic?: string;
  format?: string;
}

export const publishMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = "gcp.pubsub.publishMessage";
    const config = context.node.configuration as unknown as PublishMessageConfiguration | undefined;

    const metadata = [];
    if (config?.topic) {
      metadata.push({ icon: "message-square", label: config.topic });
    }

    return {
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "cloud",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Pub/Sub • Publish Message",
      metadata,
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as { topic?: string; messageId?: string } | undefined;

    if (context.execution.createdAt) {
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (data?.topic) {
      details["Topic"] = data.topic;
    }

    if (data?.messageId) {
      details["Message ID"] = data.messageId;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
