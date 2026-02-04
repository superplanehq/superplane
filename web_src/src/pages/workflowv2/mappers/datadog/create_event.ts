import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import { DatadogEvent } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const createEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: datadogIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default: OutputPayload[] };
    if (!outputs?.default?.[0]?.data) {
      return {};
    }
    const event = outputs.default[0].data as DatadogEvent;
    return getDetailsForEvent(event);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.alertType) {
    metadata.push({ icon: "activity", label: `Type: ${configuration.alertType}` });
  }

  if (configuration.priority) {
    metadata.push({ icon: "flag", label: `Priority: ${configuration.priority}` });
  }

  return metadata;
}

function baseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDetailsForEvent(event: DatadogEvent): Record<string, string> {
  const details: Record<string, string> = {};

  if (event?.id) {
    details["Event ID"] = String(event.id);
  }

  if (event?.title) {
    details["Title"] = event.title;
  }

  if (event?.text) {
    details["Text"] = event.text;
  }

  if (event?.date_happened) {
    details["Created At"] = new Date(event.date_happened * 1000).toLocaleString();
  }

  if (event?.alert_type) {
    details["Alert Type"] = event.alert_type;
  }

  if (event?.priority) {
    details["Priority"] = event.priority;
  }

  if (event?.tags && event.tags.length > 0) {
    details["Tags"] = event.tags.join(", ");
  }

  if (event?.url) {
    details["Event URL"] = event.url;
  }

  return details;
}
