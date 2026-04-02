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
import { buildDashboardMetadata, buildGrafanaEventSections } from "./dashboard_shared";
import type { RenderPanelOutput } from "./types";

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
      metadata: buildDashboardMetadata(context.node),
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

    if (output.dashboardUid) {
      details["Dashboard UID"] = output.dashboardUid;
    }
    if (output.panelId !== undefined) {
      details["Panel ID"] = String(output.panelId);
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
