import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
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
import { buildGrafanaEventSections } from "./alert_rule_shared";
import type { ListAlertRulesConfiguration, ListAlertRulesNodeMetadata, ListAlertRulesOutput } from "./types";

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
      metadata: buildListAlertRulesMetadata(
        context.node.configuration as ListAlertRulesConfiguration | undefined,
        context.node.metadata as ListAlertRulesNodeMetadata | undefined,
      ),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
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

function buildListAlertRulesMetadata(
  config: ListAlertRulesConfiguration | undefined,
  nodeMetadata: ListAlertRulesNodeMetadata | undefined,
): MetadataItem[] {
  const items: MetadataItem[] = [];
  const folderLabel = nodeMetadata?.folderTitle || config?.folderUID;
  if (folderLabel) items.push({ icon: "folder", label: folderLabel });
  if (config?.group) items.push({ icon: "layers-3", label: config.group });
  return items;
}
