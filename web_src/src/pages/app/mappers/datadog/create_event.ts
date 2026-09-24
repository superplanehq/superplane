import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import type { DatadogEvent } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";

export const createEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: datadogIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
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

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { alertType?: string; priority?: string } | undefined;

  if (configuration?.alertType) {
    metadata.push({ icon: "activity", label: `Type: ${configuration.alertType}` });
  }

  if (configuration?.priority) {
    metadata.push({ icon: "flag", label: `Priority: ${configuration.priority}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
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

function addDetail(details: Record<string, string>, key: string, value: string | number | undefined) {
  if (value) {
    details[key] = String(value);
  }
}

function addUnixSecondsDetail(details: Record<string, string>, key: string, value: number | undefined) {
  if (value) {
    details[key] = new Date(value * 1000).toLocaleString();
  }
}

function addJoinedDetail(details: Record<string, string>, key: string, values?: string[]) {
  if (values && values.length > 0) {
    details[key] = values.join(", ");
  }
}

function getDetailsForEvent(event: DatadogEvent): Record<string, string> {
  const details: Record<string, string> = {};

  addDetail(details, "Event ID", event.id);
  addDetail(details, "Title", event.title);
  addDetail(details, "Text", event.text);
  addUnixSecondsDetail(details, "Created At", event.date_happened);
  addDetail(details, "Alert Type", event.alert_type);
  addDetail(details, "Priority", event.priority);
  addJoinedDetail(details, "Tags", event.tags);
  addDetail(details, "Event URL", event.url);

  return details;
}
