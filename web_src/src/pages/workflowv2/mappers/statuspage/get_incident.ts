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
import statuspageIcon from "@/assets/icons/integrations/statuspage.svg";
import { StatuspageIncident, StatuspageNodeMetadata } from "./types";
import { formatTimeAgo } from "@/utils/date";
import { baseEventSections, getDetailsForIncident, truncateForDisplay } from "./utils";

export const getIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "statuspage.getIncident";

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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] };
    if (!outputs?.default || outputs.default.length === 0) {
      if (context.execution.createdAt) {
        return { "Started At": new Date(context.execution.createdAt).toLocaleString() };
      }
      return {};
    }
    const incident = outputs.default[0].data as StatuspageIncident;
    return getDetailsForIncident(incident, {
      componentName: context.node.componentName,
      execution: context.execution,
    });
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as {
    page?: string;
    incident?: string;
    incidentExpression?: string;
  };
  const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;

  const pageLabel = nodeMetadata?.pageName || configuration?.page;
  if (pageLabel) {
    metadata.push({ icon: "globe", label: "Page: " + pageLabel });
  }
  if (configuration?.incident) {
    const incidentLabel =
      configuration.incident === "__use_expression__"
        ? truncateForDisplay(configuration.incidentExpression ?? "expression")
        : (nodeMetadata?.incidentName ?? truncateForDisplay(configuration.incident));
    metadata.push({ icon: "alert-triangle", label: "Incident: " + incidentLabel });
  }

  return metadata;
}
