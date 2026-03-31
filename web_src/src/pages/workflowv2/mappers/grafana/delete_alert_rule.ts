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
import { buildAlertRuleMetadata, buildGrafanaEventSections } from "./alert_rule_shared";
import type { DeleteAlertRuleOutput } from "./types";

export const deleteAlertRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildGrafanaEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildAlertRuleMetadata(context.node, { includeUid: true }),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Deleted At": formatOptionalIsoTimestamp(context.execution.createdAt),
    };

    const response = outputs?.default?.[0]?.data as DeleteAlertRuleOutput | undefined;
    if (!response) {
      details.Response = "No data returned";
      return details;
    }

    if (response.title) {
      details.Title = response.title;
    }

    if (response.deleted) {
      details.Deleted = "Yes";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as DeleteAlertRuleOutput | undefined;
    return [response?.title, renderTimeAgo(new Date(context.execution.createdAt))].filter(Boolean).join(" · ");
  },
};
