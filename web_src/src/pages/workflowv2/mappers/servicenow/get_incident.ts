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
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { formatTimeAgo } from "@/utils/date";
import { BaseNodeMetadata } from "./types";
import { baseEventSections, buildIncidentExecutionDetails, getIncidentFromExecution, instanceUrlToLabel } from "./base";

export const getIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "servicenow";

    return {
      iconSrc: snIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    const timeAgo = formatTimeAgo(new Date(context.execution.createdAt));
    const incident = getIncidentFromExecution(context.execution);
    if (incident?.number) {
      return `${incident.number} · ${timeAgo}`;
    }
    return timeAgo;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const nodeMetadata = context.node.metadata as BaseNodeMetadata;
    return buildIncidentExecutionDetails(context.execution, nodeMetadata?.instanceUrl);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata | undefined;

  if (nodeMetadata?.instanceUrl) {
    metadata.push({ icon: "globe", label: instanceUrlToLabel(nodeMetadata.instanceUrl) });
  }

  return metadata;
}
