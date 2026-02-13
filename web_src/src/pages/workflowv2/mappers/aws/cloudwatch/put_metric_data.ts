import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface PutMetricDataConfiguration {
  region?: string;
  namespace?: string;
  metricData?: Array<{ metricName?: string }>;
}

interface PutMetricDataOutput {
  requestId?: string;
  region?: string;
  namespace?: string;
  metricCount?: number;
  metricNames?: string[];
}

export const putMetricDataMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsCloudwatchIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getPutMetricDataEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getPutMetricDataMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as PutMetricDataOutput | undefined;

    if (!result) {
      return {};
    }

    return {
      Namespace: stringOrDash(result.namespace),
      Region: stringOrDash(result.region),
      "Metric Count": stringOrDash(result.metricCount),
      "Metric Names": (result.metricNames || []).length > 0 ? (result.metricNames || []).join(", ") : "-",
      "Request ID": stringOrDash(result.requestId),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getPutMetricDataMetadata(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as PutMetricDataConfiguration | undefined;
  const metadata: MetadataItem[] = [];

  if (config?.region) {
    metadata.push({ icon: "globe", label: config.region });
  }

  if (config?.namespace) {
    metadata.push({ icon: "database", label: config.namespace });
  }

  if ((config?.metricData || []).length > 0) {
    metadata.push({
      icon: "list",
      label: `${config?.metricData?.length} metric${config?.metricData?.length === 1 ? "" : "s"}`,
    });
  }

  return metadata;
}

function getPutMetricDataEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
