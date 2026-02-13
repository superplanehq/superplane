import { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";
import { baseEventSections, getDetailsForIncident } from "./base";
import { formatTimeAgo } from "@/utils/date";

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
  const configuration = node.configuration as Record<string, unknown>;

  if (configuration?.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  const updates: string[] = [];
  if (configuration?.status) {
    updates.push(`Status: ${configuration.status}`);
  }
  if (configuration?.subStatus) {
    updates.push("Sub-Status");
  }
  if (configuration?.severity) {
    updates.push("Severity");
  }
  if (configuration?.title) {
    updates.push("Title");
  }
  if (configuration?.summary) {
    updates.push("Summary");
  }
  if (Array.isArray(configuration?.services) && (configuration.services as unknown[]).length > 0) {
    updates.push(`Services (${(configuration.services as unknown[]).length})`);
  }
  if (Array.isArray(configuration?.teams) && (configuration.teams as unknown[]).length > 0) {
    updates.push(`Teams (${(configuration.teams as unknown[]).length})`);
  }
  if (Array.isArray(configuration?.labels) && (configuration.labels as unknown[]).length > 0) {
    updates.push(`Labels (${(configuration.labels as unknown[]).length})`);
  }

  if (updates.length > 0) {
    metadata.push({ icon: "funnel", label: `Updating: ${updates.join(", ")}` });
  }

  return metadata;
}
