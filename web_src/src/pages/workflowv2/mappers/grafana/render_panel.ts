import type { ComponentBaseProps } from "@/pages/workflowv2/mappers/types";
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
import { buildDashboardSelectionMetadata, buildPanelMetadata, buildTimeRangeMetadata } from "./dashboard_shared";
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
    const defaultOutputs = context.execution.outputs?.["default"];
    const details: Record<string, string> = {
      "Rendered At": formatTimestamp(context.execution.createdAt),
    };

    const payload = (Array.isArray(defaultOutputs) ? defaultOutputs[0] : undefined) as OutputPayload | undefined;
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

    const output: RenderPanelOutput | undefined = payload.data;
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
  const configuration =
    typeof node.configuration === "object" && node.configuration !== null
      ? (node.configuration as RenderPanelConfiguration)
      : undefined;
  const nodeMetadata =
    typeof node.metadata === "object" && node.metadata !== null ? (node.metadata as DashboardNodeMetadata) : undefined;
  return [
    buildDashboardSelectionMetadata(nodeMetadata, configuration?.dashboard),
    buildPanelMetadata(nodeMetadata),
    buildTimeRangeMetadata(configuration?.from, configuration?.to),
  ].filter((item): item is NonNullable<typeof item> => Boolean(item));
}
