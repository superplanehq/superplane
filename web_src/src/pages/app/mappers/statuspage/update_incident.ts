import type { ComponentBaseProps, ComponentBaseSpec } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
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

  const typeLabel = configuration?.incidentType === "scheduled" ? "Scheduled" : "Realtime";
  values.push({
    badges: [
      { label: "Type:", bgColor: "bg-action-neutral", textColor: "text-content-secondary" },
      { label: typeLabel, bgColor: "bg-action-neutral", textColor: "text-content-primary" },
    ],
  });

  const status = configuration?.statusRealtime || configuration?.statusScheduled;
  if (status) {
    values.push({
      badges: [
        { label: "Status:", bgColor: "bg-action-neutral", textColor: "text-content-secondary" },
        { label: status, bgColor: "bg-action-neutral", textColor: "text-content-primary" },
      ],
    });
  }

  if (configuration?.body) {
    const bodyPreview = truncateForDisplay(configuration.body, 50);
    values.push({
      badges: [
        { label: "Body:", bgColor: "bg-action-neutral", textColor: "text-content-secondary" },
        { label: bodyPreview, bgColor: "bg-action-neutral", textColor: "text-content-primary" },
      ],
    });
  }
  if (configuration?.impactOverride) {
    values.push({
      badges: [
        { label: "Impact:", bgColor: "bg-action-neutral", textColor: "text-content-secondary" },
        { label: configuration.impactOverride, bgColor: "bg-action-neutral", textColor: "text-content-primary" },
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
        { label: "Components:", bgColor: "bg-action-neutral", textColor: "text-content-secondary" },
        {
          label: truncateForDisplay(componentLabel, 60),
          bgColor: "bg-action-neutral",
          textColor: "text-content-primary",
        },
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
