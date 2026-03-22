import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { BaseNodeMetadata, CreateIncidentConfiguration } from "./types";
import { URGENCY_LABELS } from "./types";
import { baseEventSections, buildIncidentExecutionDetails, instanceUrlToLabel } from "./base";

export const createIncidentMapper: ComponentBaseMapper = {
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const nodeMetadata = context.node.metadata as BaseNodeMetadata;
    return buildIncidentExecutionDetails(context.execution, nodeMetadata?.instanceUrl);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata;
  const configuration = node.configuration as CreateIncidentConfiguration;

  if (nodeMetadata?.instanceUrl) {
    metadata.push({ icon: "globe", label: instanceUrlToLabel(nodeMetadata.instanceUrl) });
  }

  if (configuration.urgency) {
    const urgencyLabel = URGENCY_LABELS[configuration.urgency] || configuration.urgency;
    metadata.push({ icon: "funnel", label: `Urgency: ${urgencyLabel}` });
  }

  return metadata;
}
