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

export const createIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "statuspage.createIncident";

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
      specs: createIncidentSpecs(context.node),
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
    name?: string;
    statusRealtime?: string;
    statusScheduled?: string;
  };
  const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;

  const pageLabel = nodeMetadata?.pageName || configuration?.page;
  if (pageLabel) {
    metadata.push({ icon: "globe", label: "Page: " + pageLabel });
  }
  if (configuration?.name) {
    metadata.push({ icon: "alert-triangle", label: configuration.name });
  }

  return metadata;
}

function createIncidentSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const configuration = node.configuration as {
    incidentType?: string;
    statusRealtime?: string;
    statusScheduled?: string;
    body?: string;
    impactOverride?: string;
    scheduledFor?: string;
    scheduledUntil?: string;
    components?: Array<{ componentId?: string; status?: string }>;
  };
  const nodeMetadata = node.metadata as StatuspageNodeMetadata | undefined;
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
  if (configuration?.scheduledFor || configuration?.scheduledUntil) {
    const scheduledLabel = [configuration.scheduledFor, configuration.scheduledUntil].filter(Boolean).join(" – ");
    values.push({
      badges: [
        { label: "Scheduled:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: truncateForDisplay(scheduledLabel, 40), bgColor: "bg-gray-100", textColor: "text-gray-800" },
      ],
    });
  }
  if (configuration?.components && configuration.components.length > 0) {
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
