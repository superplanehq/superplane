import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import newrelicIcon from "@/assets/icons/integrations/newrelic.svg";
import type { NewRelicMetricPayload, ReportMetricConfiguration } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./utils";

export const reportMetricMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: newrelicIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Reported At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs?.default?.[0]?.data) {
      return details;
    }

    const metric = outputs.default[0].data as NewRelicMetricPayload;
    return { ...details, ...getDetailsForMetric(metric) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ReportMetricConfiguration | undefined;

  if (configuration?.metricName) {
    metadata.push({ icon: "activity", label: `Metric: ${configuration.metricName}` });
  }

  if (configuration?.metricType) {
    metadata.push({ icon: "tag", label: `Type: ${configuration.metricType}` });
  }

  return metadata;
}

function getDetailsForMetric(metric: NewRelicMetricPayload): Record<string, string> {
  const details: Record<string, string> = {};

  if (metric?.metricName) {
    details["Metric Name"] = metric.metricName;
  }

  if (metric?.metricType) {
    details["Metric Type"] = metric.metricType;
  }

  if (metric?.value !== undefined) {
    details["Value"] = String(metric.value);
  }

  if (metric?.timestamp) {
    details["Timestamp"] = new Date(metric.timestamp).toLocaleString();
  }

  return details;
}
