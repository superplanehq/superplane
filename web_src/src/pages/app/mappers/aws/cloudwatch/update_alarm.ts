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
  thresholdCondition?: {
    threshold?: number;
    comparisonOperator?: string;
  };
  statistic?: string;
  period?: number;
  evaluationPeriods?: number;
  datapointsToAlarm?: number;
  alarmDescription?: string;
  treatMissingData?: string;
  unit?: string;
  actionsEnabled?: string;
  alarmActions?: string[];
  ec2Action?: string;
  okActions?: string[];
  insufficientDataActions?: string[];
}

interface UpdateAlarmNodeMetadata {
  region?: string;
  alarmName?: string;
  updatedFields?: string[];
}

const UPDATABLE_FIELDS: [keyof Configuration, string][] = [
  ["thresholdCondition", "Threshold"],
  ["statistic", "Statistic"],
  ["period", "Period"],
  ["evaluationPeriods", "Evaluation Periods"],
  ["datapointsToAlarm", "Datapoints To Alarm"],
  ["alarmDescription", "Description"],
  ["treatMissingData", "Treat Missing Data"],
  ["unit", "Unit"],
  ["actionsEnabled", "Actions Enabled"],
  ["alarmActions", "Alarm Actions"],
  ["ec2Action", "EC2 Action"],
  ["okActions", "OK Actions"],
  ["insufficientDataActions", "Insufficient Data Actions"],
];

export const updateAlarmMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildCloudWatchProps(context, updateAlarmMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const view = resolveAlarmView(context);

    return {
      "Updated At": stringOrDash(formatTimestamp(context.execution.updatedAt || context.execution.createdAt)),
      "Alarm Name": stringOrDash(view.alarmName),
      Metric: formatMetric(view.alarm?.namespace, view.alarm?.metricName),
      Condition: formatCondition(view.condition),
      Updated: view.updatedFields.length > 0 ? view.updatedFields.join(", ") : "-",
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
  condition: CloudWatchAlarm;
  updatedFields: string[];
  consoleUrl: string;
}

/** Reads the alarm from the execution output, falling back to configuration and node metadata. */
function resolveAlarmView(context: ExecutionDetailsContext): AlarmView {
  const configuration = context.node.configuration as Configuration | undefined;
  const nodeMetadata = context.node.metadata as UpdateAlarmNodeMetadata | undefined;
  const alarm = alarmFromOutputs(context.execution);

  const alarmName = firstValue(alarm?.alarmName, configuration?.alarm, nodeMetadata?.alarmName);
  const region = firstValue(alarm?.region, configuration?.region, nodeMetadata?.region);

  return {
    alarm,
    alarmName,
    condition: alarm ?? configurationCondition(configuration),
    updatedFields: inferUpdatedFields(configuration),
    consoleUrl: alarm?.consoleUrl || alarmConsoleUrl(region, alarmName),
  };
}

function updateAlarmMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as UpdateAlarmNodeMetadata | undefined;

  const metadata: MetadataItem[] = [];

  const alarmLabel = configuration?.alarm || nodeMetadata?.alarmName;
  if (alarmLabel) {
    metadata.push({ icon: "bell", label: alarmLabel });
  }

  // Derived from the live configuration so toggling fields is reflected without a publish cycle.
  for (const field of inferUpdatedFields(configuration).slice(0, 2)) {
    metadata.push({ icon: "tag", label: field });
  }

  return metadata.slice(0, MAX_METADATA_ITEMS);
}

function inferUpdatedFields(configuration: Configuration | undefined): string[] {
  if (!configuration) {
    return [];
  }

  return UPDATABLE_FIELDS.filter(([key]) => configuration[key] !== undefined && configuration[key] !== null).map(
    ([, label]) => label,
  );
}

function configurationCondition(configuration: Configuration | undefined): CloudWatchAlarm {
  return {
    statistic: configuration?.statistic,
    threshold: configuration?.thresholdCondition?.threshold,
    comparisonOperator: configuration?.thresholdCondition?.comparisonOperator,
  };
}
