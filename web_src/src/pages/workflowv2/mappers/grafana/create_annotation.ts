import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { createElement } from "react";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import type { CreateAnnotationConfiguration, CreateAnnotationOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";
import { GrafanaDashboardMetadataLabel } from "./DashboardMetadataLabel";

export const createAnnotationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as CreateAnnotationConfiguration | undefined;
    const details: Record<string, string> = {
      "Created At": formatTimestamp(context.execution.createdAt),
    };

    if (configuration?.text) {
      details["Text"] =
        configuration.text.length > 80 ? configuration.text.substring(0, 80) + "..." : configuration.text;
    }

    if (configuration?.tags && configuration.tags.length > 0) {
      details["Tags"] = configuration.tags.join(", ");
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Created At"] = payloadTimestamp;
    }

    const output = payload?.data as CreateAnnotationOutput | undefined;
    if (output != null && typeof output.id === "number") {
      details["Annotation ID"] = String(output.id);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(context: ComponentBaseContext): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = context.node.configuration as CreateAnnotationConfiguration | undefined;

  if (configuration?.text) {
    const preview = configuration.text.length > 50 ? configuration.text.substring(0, 50) + "..." : configuration.text;
    metadata.push({ icon: "bookmark", label: preview });
  }

  if (configuration?.tags && configuration.tags.length > 0) {
    metadata.push({ icon: "tag", label: configuration.tags.join(", ") });
  }

  if (configuration?.dashboardUID) {
    metadata.push({
      icon: "layout-dashboard",
      label: createElement(GrafanaDashboardMetadataLabel, {
        organizationId: context.organizationId,
        integrationId: context.integrationId,
        dashboardUid: configuration.dashboardUID,
      }),
    });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const eventTitle = title || "Trigger event";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle: eventTitle,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "-",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
