import type { ComponentBaseProps } from "@/pages/workflowv2/mappers/types";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getStateMap } from "..";
import type { ComponentBaseContext, ComponentBaseMapper, SubtitleContext } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { buildGrafanaEventSections } from "./base";
import { buildAlertRuleExecutionDetails, buildAlertRuleMetadata } from "./alert_rule_shared";

export const createAlertRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution
        ? buildGrafanaEventSections(lastExecution, componentName, { strict: true })
        : undefined,
      metadata: buildAlertRuleMetadata(context.node, { includeGroup: true, includePausedState: true }),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context) {
    return buildAlertRuleExecutionDetails(context, "Created At");
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};
