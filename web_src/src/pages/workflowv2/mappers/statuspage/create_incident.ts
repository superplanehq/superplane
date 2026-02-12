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
import statuspageIcon from "@/assets/icons/integrations/statuspage.svg";
import { StatuspageIncident } from "./types";
import { formatTimeAgo } from "@/utils/date";

function stringOrDash(value?: string | null): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return value;
}

function getDetailsForIncident(incident: StatuspageIncident): Record<string, string> {
  const details: Record<string, string> = {};

  details["ID"] = stringOrDash(incident?.id);
  details["Name"] = stringOrDash(incident?.name);
  details["Status"] = stringOrDash(incident?.status);
  details["Impact"] = stringOrDash(incident?.impact);
  details["Incident URL"] = stringOrDash(incident?.shortlink);

  if (incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  }
  if (incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  }

  const updates = incident?.incident_updates;
  if (updates && updates.length > 0) {
    const latest = updates[0];
    if (latest?.body) {
      details["Latest Update"] = latest.body;
    }
  }

  return details;
}

export const createIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "statuspage.createIncident";

    return {
      iconSrc: statuspageIcon,
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
    const outputs = context.execution.outputs as { default?: OutputPayload[] };
    if (!outputs?.default || outputs.default.length === 0) {
      if (context.execution.createdAt) {
        return { "Started At": new Date(context.execution.createdAt).toLocaleString() };
      }
      return {};
    }
    const incident = outputs.default[0].data as StatuspageIncident;
    return getDetailsForIncident(incident);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { page?: string; name?: string; incidentType?: string };

  if (configuration?.page) {
    metadata.push({ icon: "globe", label: "Page: " + configuration.page });
  }
  if (configuration?.name) {
    metadata.push({ icon: "document", label: configuration.name });
  }
  if (configuration?.incidentType) {
    metadata.push({ icon: "funnel", label: "Type: " + configuration.incidentType });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.id ?? "",
      },
    ];
  }
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  if (!rootTriggerRenderer) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.rootEvent.id,
      },
    ];
  }
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
