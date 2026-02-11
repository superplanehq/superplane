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

export interface UpdatedIncident {
  id?: string;
  sequential_id?: number;
  title?: string;
  slug?: string;
  status?: string;
  updated_at?: string;
}

export function getDetailsForUpdatedIncident(incident: UpdatedIncident): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = incident?.id || "-";
  
  if (incident?.sequential_id) {
    details["Sequential ID"] = String(incident.sequential_id);
  }
  
  details.Title = incident?.title || "-";
  details.Slug = incident?.slug || "-";
  details.Status = incident?.status || "-";

  if (incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  }

  return details;
}

export const updateIncidentMapper: ComponentBaseMapper = {
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
    const incident = outputs.default[0].data as UpdatedIncident;
    return getDetailsForUpdatedIncident(incident);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { 
    status?: string;
    severity?: string;
    incidentId?: string;
  };

  if (configuration?.incidentId) {
    metadata.push({ icon: "ticket", label: "Incident: " + configuration.incidentId });
  }

  if (configuration?.status) {
    metadata.push({ icon: "circle", label: "Status: " + configuration.status });
  }

  if (configuration?.severity) {
    metadata.push({ icon: "funnel", label: "Severity: " + configuration.severity });
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
