import { ComponentBaseProps, ComponentBaseSpec } from "@/ui/componentBase";
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

  const typeLabel = configuration?.incidentType === "scheduled" ? "Scheduled" : "Realtime";
  values.push({
    badges: [
      { label: "Type:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
      { label: typeLabel, bgColor: "bg-gray-100", textColor: "text-gray-800" },
    ],
  });

  const status = configuration?.statusRealtime || configuration?.statusScheduled;
  if (status) {
    values.push({
      badges: [
        { label: "Status:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: status, bgColor: "bg-gray-100", textColor: "text-gray-800" },
      ],
    });
  }

  if (configuration?.body) {
    const bodyPreview = truncateForDisplay(configuration.body, 50);
    values.push({
      badges: [
        { label: "Body:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: bodyPreview, bgColor: "bg-gray-100", textColor: "text-gray-800" },
      ],
    });
  }
  if (configuration?.impactOverride) {
    values.push({
      badges: [
        { label: "Impact:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: configuration.impactOverride, bgColor: "bg-gray-100", textColor: "text-gray-800" },
      ],
    });
  }
  if (configuration?.components && configuration.components.length > 0) {
    const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;
    const componentLabel =
      nodeMetadata?.componentNames && nodeMetadata.componentNames.length > 0
        ? nodeMetadata.componentNames.join(", ")
        : `${configuration.components.length} component(s)`;
    values.push({
      badges: [
        { label: "Components:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: truncateForDisplay(componentLabel, 60), bgColor: "bg-gray-100", textColor: "text-gray-800" },
      ],
    });
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
