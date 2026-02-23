import { ComponentBaseProps } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import incidentIcon from "@/assets/icons/integrations/incident.svg";
import { baseEventSections, buildIncidentExecutionDetails } from "./base";
import { formatTimeAgo } from "@/utils/date";

export const createIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: incidentIcon,
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    return buildIncidentExecutionDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { severityId?: string; visibility?: string };

  if (configuration?.visibility) {
    metadata.push({
      icon: "eye",
      label: "Visibility: " + configuration.visibility,
    });
  }

  return metadata;
}
