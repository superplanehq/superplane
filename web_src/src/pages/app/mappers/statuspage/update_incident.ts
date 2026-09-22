import type { ComponentBaseProps, ComponentBaseSpec, ComponentBaseSpecValue } from "@/ui/componentBase";
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

function updateIncidentSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const configuration = node.configuration as {
    incidentType?: string;
    statusRealtime?: string;
    statusScheduled?: string;
    body?: string;
    impactOverride?: string;
    components?: Array<{ componentId?: string; status?: string }>;
  };
  const values: ComponentBaseSpec["values"] = [];

  values.push(optionBadgeRow("Type:", incidentTypeLabel(configuration?.incidentType)));

  const status = incidentStatusLabel(configuration);
  if (status) {
    values.push(optionBadgeRow("Status:", status));
  }

  if (configuration?.body) {
    values.push(optionBadgeRow("Body:", truncateForDisplay(configuration.body, 50)));
  }
  if (configuration?.impactOverride) {
    values.push(optionBadgeRow("Impact:", configuration.impactOverride));
  }
  if (configuration?.components && configuration.components.length > 0) {
    const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;
    const componentLabel =
      nodeMetadata?.componentNames && nodeMetadata.componentNames.length > 0
        ? nodeMetadata.componentNames.join(", ")
        : `${configuration.components.length} component(s)`;
    values.push(optionBadgeRow("Components:", truncateForDisplay(componentLabel, 60)));
  }

  return [
    {
      title: "option",
      tooltipTitle: "Configuration options",
      iconSlug: "settings",
      values,
    },
  ];
}

function incidentTypeLabel(incidentType?: string): string {
  if (incidentType === "scheduled") {
    return "Scheduled";
  }

  return "Realtime";
}

function incidentStatusLabel(configuration: { statusRealtime?: string; statusScheduled?: string }): string | undefined {
  return configuration?.statusRealtime || configuration?.statusScheduled;
}

function optionBadgeRow(label: string, value: string): ComponentBaseSpecValue {
  return {
    badges: [
      { label, bgColor: "bg-gray-100", textColor: "text-gray-700" },
      { label: value, bgColor: "bg-gray-100", textColor: "text-gray-800" },
    ],
  };
}
