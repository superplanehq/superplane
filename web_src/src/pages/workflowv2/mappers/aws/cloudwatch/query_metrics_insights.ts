import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { numberOrZero, stringOrDash } from "../../utils";
import { CloudWatchMetricsInsightsOutput } from "./types";

interface QueryMetricsInsightsConfiguration {
  region?: string;
  query?: string;
  lookbackMinutes?: number;
  maxDatapoints?: number;
  scanBy?: string;
}

export const queryMetricsInsightsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsCloudwatchIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? queryMetricsInsightsEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: queryMetricsInsightsMetadataList(context.node),
      specs: queryMetricsInsightsSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CloudWatchMetricsInsightsOutput | undefined;
    if (!result) {
      return {};
    }

    const datapoints = (result.results || []).reduce(
      (count, metric) => count + numberOrZero(metric.values?.length),
      0,
    );

    return {
      Region: stringOrDash(result.region),
      "Request ID": stringOrDash(result.requestId),
      "Start Time": result.startTime ? formatTimestampInUserTimezone(result.startTime) : "-",
      "End Time": result.endTime ? formatTimestampInUserTimezone(result.endTime) : "-",
      "Result Series": numberOrZero(result.results?.length).toString(),
      Datapoints: datapoints.toString(),
      Messages: numberOrZero(result.messages?.length).toString(),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function queryMetricsInsightsMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryMetricsInsightsConfiguration | undefined;

  if (configuration?.region) {
    metadata.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.lookbackMinutes) {
    metadata.push({ icon: "clock", label: `Last ${configuration.lookbackMinutes}m` });
  }

  if (configuration?.scanBy) {
    metadata.push({ icon: "arrow-up-down", label: configuration.scanBy });
  }

  return metadata;
}

function queryMetricsInsightsSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as QueryMetricsInsightsConfiguration | undefined;

  if (configuration?.query) {
    specs.push({
      title: "query",
      tooltipTitle: "Metrics Insights query",
      iconSlug: "search",
      value: configuration.query,
      contentType: "text",
    });
  }

  return specs;
}

function queryMetricsInsightsEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id!,
    },
  ];
}
