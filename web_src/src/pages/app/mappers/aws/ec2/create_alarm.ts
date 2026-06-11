import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import type { Ec2Alarm } from "./types";

interface Configuration {
  region?: string;
  instance?: string;
  alarmName?: string;
  metricName?: string;
  statistic?: string;
  comparisonOperator?: string;
  threshold?: number;
  alarmAction?: string;
  snsTopic?: string;
}

interface CreateAlarmNodeMetadata {
  region?: string;
  instanceId?: string;
  instanceName?: string;
  alarmName?: string;
  metricName?: string;
}

export const createAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? createAlarmEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: createAlarmMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const nodeMetadata = context.node.metadata as CreateAlarmNodeMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Ec2Alarm | undefined;
    const createdAt = formatTimestamp(context.execution.createdAt);
    if (!output) {
      return createAlarmDetailsFallback(configuration, nodeMetadata, createdAt);
    }
    return createAlarmDetailsFromOutput(output, configuration, createdAt);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function createAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as CreateAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = configuration?.alarmName || nodeMetadata?.alarmName;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  const instanceLabel = nodeMetadata?.instanceName || nodeMetadata?.instanceId || configuration?.instance;
  if (instanceLabel) {
    metadata.push({ icon: "server", label: instanceLabel });
  }

  const metricLabel = nodeMetadata?.metricName || configuration?.metricName;
  if (metricLabel) {
    metadata.push({ icon: "chart-line", label: metricLabel });
  }

  return metadata.slice(0, 3);
}

function formatTimestamp(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

function alarmActionExtras(configuration?: Configuration): Record<string, string> {
  const extras: Record<string, string> = {};
  if (configuration?.alarmAction) {
    extras["Alarm Action"] = configuration.alarmAction;
  }
  if (configuration?.snsTopic) {
    extras["SNS Topic"] = configuration.snsTopic;
  }
  return extras;
}

function createAlarmDetailsFallback(
  configuration: Configuration | undefined,
  nodeMetadata: CreateAlarmNodeMetadata | undefined,
  createdAt: string | undefined,
): Record<string, string> {
  return {
    "Created At": stringOrDash(createdAt),
    "Alarm Name": stringOrDash(configuration?.alarmName ?? nodeMetadata?.alarmName),
    Metric: stringOrDash(configuration?.metricName),
    Threshold: "-",
    State: "-",
    Region: stringOrDash(configuration?.region ?? nodeMetadata?.region),
    ...alarmActionExtras(configuration),
  };
}

function createAlarmDetailsFromOutput(
  output: Ec2Alarm,
  configuration: Configuration | undefined,
  createdAt: string | undefined,
): Record<string, string> {
  return {
    "Created At": stringOrDash(createdAt),
    "Alarm Name": stringOrDash(output.alarmName ?? configuration?.alarmName),
    Metric: stringOrDash(output.metricName ?? configuration?.metricName),
    Threshold: output.threshold !== undefined ? String(output.threshold) : "-",
    State: stringOrDash(output.stateValue),
    Region: stringOrDash(output.region ?? configuration?.region),
    ...alarmActionExtras(configuration),
  };
}

function createAlarmEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
