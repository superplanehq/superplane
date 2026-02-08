import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
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
import { getDetailsForIncident } from "./base";
import { Incident } from "./types";

interface UpdateIncidentConfiguration {
  incidentId?: string;
  title?: string;
  summary?: string;
  status?: string;
  severity?: string;
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
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs?.default || outputs.default.length === 0) {
      return {};
    }

    const incident = outputs.default[0].data as Incident;
    return getDetailsForIncident(incident);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateIncidentConfiguration | undefined;

  if (configuration?.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  const updates: string[] = [];
  if (configuration?.status) {
    updates.push(`Status: ${configuration.status}`);
  }
  if (configuration?.severity) {
    updates.push(`Severity: ${configuration.severity}`);
  }
  if (configuration?.title) {
    updates.push("Title");
  }
  if (configuration?.summary) {
    updates.push("Summary");
  }

  if (updates.length > 0) {
    metadata.push({ icon: "funnel", label: `Updating: ${updates.join(", ")}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

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
