import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
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
import { MetadataItem } from "@/ui/metadataList";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { formatTimeAgo } from "@/utils/date";

export interface IncidentEvent {
  id?: string;
  event?: string;
  visibility?: string;
  occurred_at?: string;
  created_at?: string;
}

export const createEventMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: rootlyIcon,
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
    if (!outputs?.default || outputs.default.length === 0) {
      return {};
    }
    const event = outputs.default[0].data as IncidentEvent;
    return getDetailsForEvent(event);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getDetailsForEvent(event: IncidentEvent): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = event?.id || "-";
  details.Event = event?.event || "-";
  details.Visibility = event?.visibility || "-";

  if (event?.occurred_at) {
    details["Occurred At"] = new Date(event.occurred_at).toLocaleString();
  }

  if (event?.created_at) {
    details["Created At"] = new Date(event.created_at).toLocaleString();
  }

  return details;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { visibility?: string; incidentId?: string };

  if (configuration?.visibility) {
    metadata.push({ icon: "eye", label: "Visibility: " + configuration.visibility });
  }

  if (configuration?.incidentId) {
    metadata.push({ icon: "hashtag", label: "Incident: " + configuration.incidentId });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
