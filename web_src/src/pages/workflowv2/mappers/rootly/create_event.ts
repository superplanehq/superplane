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
import { IncidentEvent } from "./types";
import { baseEventSections, getDetailsForIncidentEvent } from "./base";
import { formatTimeAgo } from "@/utils/date";

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
    const incidentEvent = outputs.default[0].data as IncidentEvent;
    return getDetailsForIncidentEvent(incidentEvent);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { incidentId?: string; visibility?: string };

  if (configuration?.incidentId) {
    metadata.push({ icon: "hash", label: "Incident: " + configuration.incidentId });
  }

  if (configuration?.visibility) {
    metadata.push({ icon: "eye", label: "Visibility: " + configuration.visibility });
  }

  return metadata;
}
