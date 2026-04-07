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
import { buildGrafanaEventSections } from "./dashboard_shared";
import type { SearchDashboardsOutput } from "./types";

export const searchDashboardsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildGrafanaEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: [],
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
    };

    const response = outputs?.default?.[0]?.data as SearchDashboardsOutput | undefined;
    if (!response) {
      details.Response = "No data returned";
      return details;
    }

    const dashboards = response?.dashboards ?? [];

    details["Dashboard Count"] = String(dashboards.length);
    if (dashboards[0]?.title) {
      details["First Dashboard"] = dashboards[0].title;
    }
    if (dashboards.length > 1 && dashboards[1]?.title) {
      details["Second Dashboard"] = dashboards[1].title;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as SearchDashboardsOutput | undefined;
    const count = response?.dashboards?.length ?? 0;
    return [`${count} dashboard${count === 1 ? "" : "s"}`, renderTimeAgo(new Date(context.execution.createdAt))]
      .filter(Boolean)
      .join(" · ");
  },
};
