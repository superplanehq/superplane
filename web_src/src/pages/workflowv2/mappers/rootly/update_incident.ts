import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/time";
import { getBackgroundColorClass } from "@/ui/utils";
import { baseEventSections, getDetailsForIncident, getStateMap, Incident } from "./base";
import rootlyIcon from "@/assets/rootly-icon.svg";

interface UpdateIncidentConfiguration {
  incidentId?: string;
  title?: string;
  summary?: string;
  status?: string;
  severity?: string;
}

function getUpdateIncidentMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateIncidentConfiguration | undefined;

  if (configuration?.status) {
    metadata.push({ icon: "activity", label: "→ " + configuration.status });
  }

  if (configuration?.severity) {
    metadata.push({ icon: "funnel", label: "Severity: " + configuration.severity });
  }

  return metadata;
}

export const updateIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution
        ? baseEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: getUpdateIncidentMetadataList(context.node),
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
    const details = getDetailsForIncident(incident);

    // Add update-specific details
    const config = context.execution.configuration as UpdateIncidentConfiguration | undefined;
    if (config?.status) {
      details["Updated Status"] = config.status;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};