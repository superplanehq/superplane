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
  alarm?: string;
  thresholdCondition?: {
    threshold?: number;
    comparisonOperator?: string;
  };
  statistic?: string;
  period?: number;
  evaluationPeriods?: number;
  alarmDescription?: string;
  treatMissingData?: string;
  alarmAction?: string;
  snsTopic?: string;
}

interface UpdateAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
  updatedFields?: string[];
}

export const updateAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? updateAlarmEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: updateAlarmMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const nodeMetadata = context.node.metadata as UpdateAlarmNodeMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as Ec2Alarm | undefined;
    const updatedAt = formatTimestamp(context.execution.updatedAt || context.execution.createdAt);
    if (!output) {
      return updateAlarmDetailsFallback(configuration, nodeMetadata, updatedAt);
    }
    return updateAlarmDetailsFromOutput(output, configuration, updatedAt);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function updateAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as UpdateAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = nodeMetadata?.alarmName || configuration?.alarm;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  // Always derive from the live configuration so toggling fields on/off is
  // immediately reflected without requiring a publish cycle.
  for (const field of inferUpdatedFields(configuration).slice(0, 2)) {
    metadata.push({ icon: "pen-line", label: field });
  }

  return metadata.slice(0, 3);
}

function inferUpdatedFields(configuration: Configuration | undefined): string[] {
  if (!configuration) {
    return [];
  }

  const fieldMap: [keyof Configuration, string][] = [
    ["thresholdCondition", "Threshold"],
    ["statistic", "Statistic"],
    ["period", "Period"],
    ["evaluationPeriods", "Evaluation Periods"],
    ["alarmDescription", "Description"],
    ["treatMissingData", "Treat Missing Data"],
    ["alarmAction", "Alarm Action"],
    ["snsTopic", "SNS Topic"],
  ];

  return fieldMap
    .filter(([key]) => configuration[key] !== undefined && configuration[key] !== null)
    .map(([, label]) => label);
}

function formatTimestamp(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

function updateAlarmDetailsFallback(
  configuration: Configuration | undefined,
  nodeMetadata: UpdateAlarmNodeMetadata | undefined,
  updatedAt: string | undefined,
): Record<string, string> {
  return {
    "Updated At": stringOrDash(updatedAt),
    "Alarm Name": stringOrDash(configuration?.alarm ?? nodeMetadata?.alarmName),
    Metric: "-",
    Threshold:
      configuration?.thresholdCondition?.threshold !== undefined
        ? String(configuration.thresholdCondition.threshold)
        : "-",
    State: "-",
    Region: stringOrDash(configuration?.region ?? nodeMetadata?.region),
  };
}

function updateAlarmDetailsFromOutput(
  output: Ec2Alarm,
  configuration: Configuration | undefined,
  updatedAt: string | undefined,
): Record<string, string> {
  return {
    "Updated At": stringOrDash(updatedAt),
    "Alarm Name": stringOrDash(output.alarmName ?? configuration?.alarm),
    Metric: stringOrDash(output.metricName),
    Threshold: output.threshold !== undefined ? String(output.threshold) : "-",
    State: stringOrDash(output.stateValue),
    Region: stringOrDash(output.region ?? configuration?.region),
  };
}

function updateAlarmEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
