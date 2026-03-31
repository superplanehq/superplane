import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatOptionalIsoTimestamp } from "@/lib/timezone";
import { buildGrafanaEventSections } from "./alert_rule_shared";
import type { ListAlertRulesOutput } from "./types";

export const listAlertRulesMapper: ComponentBaseMapper = {
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
      "Listed At": formatOptionalIsoTimestamp(context.execution.createdAt),
    };

    const response = outputs?.default?.[0]?.data as ListAlertRulesOutput | undefined;
    if (!response) {
      details.Response = "No data returned";
      return details;
    }

    const alertRules = response?.alertRules ?? [];

    details["Alert Count"] = String(alertRules.length);
    if (alertRules[0]?.title) {
      details["First Alert Rule"] = alertRules[0].title;
    }
    if (alertRules.length > 1 && alertRules[1]?.title) {
      details["Second Alert Rule"] = alertRules[1].title;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as ListAlertRulesOutput | undefined;
    const count = response?.alertRules?.length ?? 0;
    return [`${count} rule${count === 1 ? "" : "s"}`, renderTimeAgo(new Date(context.execution.createdAt))]
      .filter(Boolean)
      .join(" · ");
  },
};
