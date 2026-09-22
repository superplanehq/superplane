import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import type { CheckRuleNodeMetadata, CheckRulePayload, GetCheckRuleConfiguration } from "./types";
import { truncate } from "../safeMappers";
import { renderTimeAgo } from "@/components/TimeAgo";

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

    if (!hasDefaultPayload(outputs)) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as CheckRulePayload | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};
    const fetchedAt = payload.timestamp;
    const name = responseData.name;
    const id = responseData.id;
    const expression = responseData.expression;
    const interval = responseData.interval;
    const forDuration = responseData.for;
    const keepFiringFor = responseData.keepFiringFor;
    const enabled = responseData.enabled;

    if (fetchedAt) details["Fetched At"] = new Date(fetchedAt).toLocaleString();
    if (name) details["Name"] = String(name);
    if (id) details["ID"] = String(id);

    if (expression) {
      const expr = String(expression);
      details["Expression"] = expr.length > 100 ? expr.substring(0, 100) + "..." : expr;
    }

    addCheckRuleThresholds(details, responseData);

    if (interval) details["Interval"] = String(interval);
    if (forDuration) details["For"] = String(forDuration);
    if (keepFiringFor) details["Keep Firing For"] = String(keepFiringFor);
    if (enabled != null) details["Enabled"] = enabled ? "Yes" : "No";

    addCheckRuleLabels(details, responseData);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function hasDefaultPayload(
  outputs: { default?: OutputPayload[] } | undefined,
): outputs is { default: OutputPayload[] } {
  return !!outputs?.default && outputs.default.length > 0;
}

function addCheckRuleThresholds(details: Record<string, string>, responseData: CheckRulePayload) {
  if (responseData.thresholds) {
    const parts: string[] = [];
    if (responseData.thresholds.degraded != null) parts.push(`Degraded: ${responseData.thresholds.degraded}`);
    if (responseData.thresholds.critical != null) parts.push(`Critical: ${responseData.thresholds.critical}`);
    if (parts.length > 0) details["Thresholds"] = parts.join(", ");
  }
}

function addCheckRuleLabels(details: Record<string, string>, responseData: CheckRulePayload) {
  if (responseData.labels && Object.keys(responseData.labels).length > 0) {
    const labels = Object.entries(responseData.labels)
      .map(([k, v]) => `${k}=${v}`)
      .join(", ");
    details["Labels"] = labels;
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as CheckRuleNodeMetadata | undefined;
  const configuration = node.configuration as GetCheckRuleConfiguration;

  if (nodeMetadata?.checkRuleName) {
    metadata.push({ icon: "bell", label: nodeMetadata.checkRuleName });
  } else if (configuration?.checkRule) {
    const idPreview = truncate(configuration.checkRule, 30, "…");
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
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
