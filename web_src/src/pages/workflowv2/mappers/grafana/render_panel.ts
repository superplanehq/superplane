import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getStateMap } from "..";
import { renderTimeAgo } from "@/components/TimeAgo";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { formatTimestamp } from "../utils";
import { buildGrafanaEventSections } from "./base";
import {
  buildDashboardSelectionMetadata,
  buildPanelMetadata,
  buildTimeRangeMetadata,
  previewMetadataItem,
} from "./dashboard_shared";
import type { DashboardNodeMetadata, RenderPanelConfiguration, RenderPanelOutput } from "./types";

export const renderPanelMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildGrafanaEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Rendered At": formatTimestamp(context.execution.createdAt),
    };

    const payload = outputs?.default?.[0];
    if (!payload) {
      details.Response = "No data returned";
      return details;
    }

    if (payload.timestamp) {
      const ts = formatTimestamp(payload.timestamp);
      if (ts !== "-") {
        details["Rendered At"] = ts;
      }
    }

    const output = payload.data as RenderPanelOutput | undefined;
    if (!output) {
      details.Response = "No data returned";
      return details;
    }

    if (output.dashboard) {
      details.Dashboard = output.dashboard;
    }
    if (output.panel !== undefined) {
      details.Panel = String(output.panel);
    }
    if (output.url) {
      details.URL = output.url;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function buildMetadata(node: ComponentBaseContext["node"]) {
  const configuration = node.configuration as RenderPanelConfiguration | undefined;
  const nodeMetadata = node.metadata as DashboardNodeMetadata | undefined;
  const size =
    configuration?.width && configuration?.height ? `${configuration.width}x${configuration.height}` : undefined;

  return [
    buildDashboardSelectionMetadata(nodeMetadata, configuration?.dashboard),
    buildPanelMetadata(nodeMetadata),
    buildTimeRangeMetadata(configuration?.from, configuration?.to),
    previewMetadataItem("maximize", "Size: ", size),
  ]
    .filter((item): item is NonNullable<typeof item> => Boolean(item))
    .slice(0, 3);
}
