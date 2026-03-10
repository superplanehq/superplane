import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { CheckRuleNodeMetadata, GetCheckRuleConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const getCheckRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, any> | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};

    if (payload?.timestamp) details["Fetched At"] = new Date(payload.timestamp).toLocaleString();
    if (responseData.name) details["Name"] = String(responseData.name);
    if (responseData.id) details["ID"] = String(responseData.id);

    if (responseData.expression) {
      const expr = String(responseData.expression);
      details["Expression"] = expr.length > 100 ? expr.substring(0, 100) + "..." : expr;
    }

    if (responseData.thresholds) {
      const parts: string[] = [];
      if (responseData.thresholds.degraded != null) parts.push(`Degraded: ${responseData.thresholds.degraded}`);
      if (responseData.thresholds.critical != null) parts.push(`Critical: ${responseData.thresholds.critical}`);
      if (parts.length > 0) details["Thresholds"] = parts.join(", ");
    }

    if (responseData.interval) details["Interval"] = String(responseData.interval);
    if (responseData.for) details["For"] = String(responseData.for);
    if (responseData.keepFiringFor) details["Keep Firing For"] = String(responseData.keepFiringFor);
    if (responseData.enabled != null) details["Enabled"] = responseData.enabled ? "Yes" : "No";

    if (responseData.labels && Object.keys(responseData.labels).length > 0) {
      const labels = Object.entries(responseData.labels)
        .map(([k, v]) => `${k}=${v}`)
        .join(", ");
      details["Labels"] = labels;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as CheckRuleNodeMetadata | undefined;
  const configuration = node.configuration as GetCheckRuleConfiguration;

  if (nodeMetadata?.checkRuleName) {
    metadata.push({ icon: "bell", label: nodeMetadata.checkRuleName });
  } else if (configuration?.checkRule) {
    const idPreview =
      configuration.checkRule.length > 30 ? configuration.checkRule.substring(0, 30) + "…" : configuration.checkRule;
    metadata.push({ icon: "bell", label: idPreview });
  }

  if (configuration?.dataset) metadata.push({ icon: "database", label: configuration.dataset });
  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent || !execution.createdAt || !execution.rootEvent.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !rootTriggerNode.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
