import type { ComponentBaseProps, ComponentBaseSpec } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "../mapperLookup";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import statuspageIcon from "@/assets/icons/integrations/statuspage.svg";
import type { StatuspageIncident, StatuspageNodeMetadata } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections, getDetailsForIncident, truncateForDisplay } from "./utils";

export const updateIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "statuspage.updateIncident";

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
      specs: updateIncidentSpecs(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
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

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as {
    page?: string;
    incident?: string;
    incidentExpression?: string;
    statusRealtime?: string;
    statusScheduled?: string;
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

interface UpdateIncidentConfiguration {
  incidentType?: string;
  statusRealtime?: string;
  statusScheduled?: string;
  body?: string;
  impactOverride?: string;
  components?: Array<{ componentId?: string; status?: string }>;
}

function updateIncidentSpecs(node: NodeInfo): ComponentBaseSpec[] {
  return [
    {
      title: "option",
      tooltipTitle: "Configuration options",
      iconSlug: "settings",
      values: updateIncidentSpecValues(node),
    },
  ];
}

function updateIncidentSpecValues(node: NodeInfo): ComponentBaseSpec["values"] {
  const configuration = node.configuration as UpdateIncidentConfiguration | undefined;
  const values: NonNullable<ComponentBaseSpec["values"]> = [];

  pushSpecBadge(values, "Type:", incidentTypeLabel(configuration?.incidentType));
  pushSpecBadge(values, "Status:", incidentStatusLabel(configuration));
  pushSpecBadge(values, "Body:", incidentBodyLabel(configuration));
  pushSpecBadge(values, "Impact:", configuration?.impactOverride);
  pushSpecBadge(values, "Components:", incidentComponentLabel(node, configuration));

  return values;
}

function incidentTypeLabel(incidentType?: string): string {
  return incidentType === "scheduled" ? "Scheduled" : "Realtime";
}

function incidentStatusLabel(configuration: UpdateIncidentConfiguration | undefined): string | undefined {
  return configuration?.statusRealtime || configuration?.statusScheduled;
}

function incidentBodyLabel(configuration: UpdateIncidentConfiguration | undefined): string | undefined {
  if (!configuration?.body) {
    return undefined;
  }

  return truncateForDisplay(configuration.body, 50);
}

function incidentComponentLabel(
  node: NodeInfo,
  configuration: UpdateIncidentConfiguration | undefined,
): string | undefined {
  if (!configuration?.components || configuration.components.length === 0) {
    return undefined;
  }

  const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;
  const componentLabel =
    nodeMetadata?.componentNames && nodeMetadata.componentNames.length > 0
      ? nodeMetadata.componentNames.join(", ")
      : `${configuration.components.length} component(s)`;

  return truncateForDisplay(componentLabel, 60);
}

function pushSpecBadge(values: NonNullable<ComponentBaseSpec["values"]>, label: string, value?: string) {
  if (!value) {
    return;
  }

  values.push({
    badges: [
      { label, bgColor: "bg-gray-100", textColor: "text-gray-700" },
      { label: value, bgColor: "bg-gray-100", textColor: "text-gray-800" },
    ],
  });
}
