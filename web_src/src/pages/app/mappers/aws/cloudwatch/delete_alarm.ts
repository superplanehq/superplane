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
  alarmFromOutputs,
  buildCloudWatchProps,
  cloudwatchSubtitle,
  firstValue,
  formatCondition,
  formatMetric,
  formatTimestamp,
  MAX_METADATA_ITEMS,
} from "./common";
import type { CloudWatchDeletedAlarm } from "./types";

interface Configuration {
  region?: string;
  alarm?: string;
}

interface DeleteAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
}

export const deleteAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildCloudWatchProps(context, deleteAlarmMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const nodeMetadata = context.node.metadata as DeleteAlarmNodeMetadata | undefined;
    const alarm = alarmFromOutputs(context.execution) as CloudWatchDeletedAlarm | undefined;

    return {
      "Deleted At": stringOrDash(formatTimestamp(context.execution.updatedAt || context.execution.createdAt)),
      "Alarm Name": stringOrDash(firstValue(alarm?.alarmName, configuration?.alarm, nodeMetadata?.alarmName)),
      Metric: formatMetric(alarm?.namespace, alarm?.metricName),
      Condition: formatCondition(alarm),
      Region: stringOrDash(firstValue(alarm?.region, configuration?.region, nodeMetadata?.region)),
      Deleted: alarm?.deleted === true ? "Yes" : "-",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return cloudwatchSubtitle(context);
  },
};

function deleteAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as DeleteAlarmNodeMetadata | undefined;

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
