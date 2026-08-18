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
  alarm?: string;
}

interface GetAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
}

export const getAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildCloudWatchProps(context, getAlarmMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const view = resolveAlarmView(context);

    return {
      "Retrieved At": stringOrDash(formatTimestamp(context.execution.updatedAt || context.execution.createdAt)),
      "Alarm Name": stringOrDash(view.alarmName),
      Metric: formatMetric(view.alarm?.namespace, view.alarm?.metricName),
      Condition: formatCondition(view.alarm),
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
  consoleUrl: string;
}

/** Reads the alarm from the execution output, falling back to configuration and node metadata. */
function resolveAlarmView(context: ExecutionDetailsContext): AlarmView {
  const configuration = context.node.configuration as Configuration | undefined;
  const nodeMetadata = context.node.metadata as GetAlarmNodeMetadata | undefined;
  const alarm = alarmFromOutputs(context.execution);

  const alarmName = firstValue(alarm?.alarmName, configuration?.alarm, nodeMetadata?.alarmName);
  const region = firstValue(alarm?.region, configuration?.region, nodeMetadata?.region);

  return {
    alarm,
    alarmName,
    consoleUrl: alarm?.consoleUrl || alarmConsoleUrl(region, alarmName),
  };
}

function getAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as GetAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = configuration?.alarm || nodeMetadata?.alarmName;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  const region = configuration?.region || nodeMetadata?.region;
  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata.slice(0, MAX_METADATA_ITEMS);
}
