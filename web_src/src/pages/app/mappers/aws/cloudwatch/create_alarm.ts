import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import {
  alarmConsoleUrl,
  alarmFromOutputs,
  buildCloudWatchProps,
  cloudwatchSubtitle,
  firstValue,
  formatCondition,
  formatMetric,
  formatTimestamp,
  MAX_METADATA_ITEMS,
} from "./common";
import type { CloudWatchAlarm } from "./types";

interface Configuration {
  region?: string;
  alarmName?: string;
  namespace?: string;
  metricName?: string;
  statistic?: string;
  comparisonOperator?: string;
  threshold?: number;
}

interface CreateAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
  namespace?: string;
  metricName?: string;
  dimensions?: string;
}

export const createAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildCloudWatchProps(context, createAlarmMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const view = resolveAlarmView(context);

    return {
      "Created At": stringOrDash(formatTimestamp(context.execution.createdAt)),
      "Alarm Name": stringOrDash(view.alarmName),
      Metric: formatMetric(view.namespace, view.metricName),
      Condition: formatCondition(view.condition),
      State: stringOrDash(view.alarm?.stateValue),
      "CloudWatch Console": stringOrDash(view.consoleUrl),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return cloudwatchSubtitle(context);
  },
};

interface AlarmView {
  alarm?: CloudWatchAlarm;
  alarmName?: string;
  namespace?: string;
  metricName?: string;
  condition?: CloudWatchAlarm;
  consoleUrl: string;
}

/** Reads the alarm from the execution output, falling back to configuration and node metadata. */
function resolveAlarmView(context: ExecutionDetailsContext): AlarmView {
  const configuration = context.node.configuration as Configuration | undefined;
  const nodeMetadata = context.node.metadata as CreateAlarmNodeMetadata | undefined;
  const alarm = alarmFromOutputs(context.execution);

  const alarmName = firstValue(alarm?.alarmName, configuration?.alarmName, nodeMetadata?.alarmName);
  const region = firstValue(alarm?.region, configuration?.region, nodeMetadata?.region);

  return {
    alarm,
    alarmName,
    namespace: firstValue(alarm?.namespace, configuration?.namespace),
    metricName: firstValue(alarm?.metricName, configuration?.metricName),
    condition: alarm ?? configuration,
    consoleUrl: alarm?.consoleUrl || alarmConsoleUrl(region, alarmName),
  };
}

function createAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as CreateAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = configuration?.alarmName || nodeMetadata?.alarmName;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  const metricLabel = configuration?.metricName || nodeMetadata?.metricName;
  if (metricLabel) {
    metadata.push({ icon: "chart-line", label: metricLabel });
  }

  const dimensionsLabel = nodeMetadata?.dimensions;
  if (dimensionsLabel) {
    metadata.push({ icon: "tag", label: dimensionsLabel });
  }

  return metadata.slice(0, MAX_METADATA_ITEMS);
}
