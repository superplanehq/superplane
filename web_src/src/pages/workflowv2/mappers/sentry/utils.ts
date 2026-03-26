import type { EventSection } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import type {
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
  TriggerRenderer,
} from "../types";

export function buildEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  getTriggerRenderer: (name: string) => TriggerRenderer,
  getState: (componentName: string) => StateFunction,
): EventSection[] | undefined {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  const rootComponentName = rootTriggerNode?.componentName;

  if (!rootEvent || !createdAt || !rootComponentName) {
    return undefined;
  }

  const rootTriggerRenderer = getTriggerRenderer(rootComponentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id || "",
    },
  ];
}

export type SentryAlertRuleTrigger = {
  label?: string;
  alertThreshold?: number;
};

export type SentryAlertRule = {
  name?: string;
  environment?: string | null;
  projects?: string[];
  query?: string;
  aggregate?: string;
  triggers?: SentryAlertRuleTrigger[];
};

export type AlertRuleNodeMetadata = {
  project?: {
    name?: string;
    slug?: string;
  };
  alertName?: string;
};

export type SentryAlertThresholdConfiguration = {
  threshold?: number;
  notification?: {
    targetType?: string;
  };
};

export function getSentryMetricAlertRuleFromDefaultOutput(outputs: unknown): SentryAlertRule | undefined {
  const bucket = outputs as { default?: OutputPayload[] } | undefined;
  return bucket?.default?.[0]?.data as SentryAlertRule | undefined;
}

export function subtitleForSentryMetricAlertRule(context: SubtitleContext): string {
  const alertRule = getSentryMetricAlertRuleFromDefaultOutput(context.execution.outputs);
  const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
  return [alertRule?.name, timestamp].filter(Boolean).join(" · ");
}

export function executionDetailsForSentryMetricAlertRule(context: ExecutionDetailsContext): Record<string, string> {
  const alertRule = getSentryMetricAlertRuleFromDefaultOutput(context.execution.outputs);
  const details: Record<string, string> = {};

  addFormattedTimestamp(details, "Started At", context.execution.createdAt);
  addOrderedDetails(details, [
    { label: "Name", value: alertRule?.name },
    { label: "Project", value: alertRule?.projects?.[0] },
    { label: "Environment", value: alertRule?.environment || undefined },
    { label: "Aggregate", value: alertRule?.aggregate },
    { label: "Query", value: alertRule?.query || undefined },
    { label: "Triggers", value: summarizeTriggers(alertRule) },
  ]);

  return details;
}

export function summarizeTriggers(alertRule: SentryAlertRule | undefined): string | undefined {
  if (!alertRule?.triggers?.length) {
    return undefined;
  }

  return alertRule.triggers
    .map((trigger) => {
      if (!trigger.label) {
        return undefined;
      }

      if (trigger.alertThreshold === undefined) {
        return trigger.label;
      }

      return `${trigger.label}: ${trigger.alertThreshold}`;
    })
    .filter(Boolean)
    .join(", ");
}

export function getAlertRuleProjectLabel(
  nodeMetadata: AlertRuleNodeMetadata | undefined,
  configuration: { project?: string } | undefined,
): string | undefined {
  return nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
}

export function getAlertRuleSelectionLabel(
  nodeMetadata: AlertRuleNodeMetadata | undefined,
  configuration: { alertId?: string } | undefined,
): string | undefined {
  return nodeMetadata?.alertName || configuration?.alertId;
}

export function getAlertThresholdMetadataLabel(
  configuration:
    | {
        thresholdType?: string;
        critical?: { threshold?: number };
        warning?: { threshold?: number };
      }
    | undefined,
): string | undefined {
  const comparison = configuration?.thresholdType === "below" ? "≤" : "≥";

  if (configuration?.critical?.threshold !== undefined) {
    return `Critical ${comparison} ${configuration.critical.threshold}`;
  }

  if (configuration?.warning?.threshold !== undefined) {
    return `Warning ${comparison} ${configuration.warning.threshold}`;
  }

  return undefined;
}

export function splitSentryIssueTitle(title?: string): { title?: string; prefix?: string } {
  if (!title) {
    return {};
  }

  const trimmedTitle = title.trim();
  if (!trimmedTitle) {
    return {};
  }

  const separatorIndex = trimmedTitle.indexOf(":");
  if (separatorIndex <= 0 || separatorIndex >= trimmedTitle.length - 1) {
    return { title: trimmedTitle };
  }

  const prefix = trimmedTitle.slice(0, separatorIndex).trim();
  const suffix = trimmedTitle.slice(separatorIndex + 1).trim();

  if (!prefix || !suffix) {
    return { title: trimmedTitle };
  }

  return {
    title: suffix,
    prefix,
  };
}

export function addDetail(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = value;
}

export function addFormattedTimestamp(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = new Date(value).toLocaleString();
}

export interface OrderedDetail {
  label: string;
  value?: string;
  isTimestamp?: boolean;
}

export function addOrderedDetails(details: Record<string, string>, orderedDetails: OrderedDetail[], maxItems = 6) {
  for (const detail of orderedDetails) {
    if (Object.keys(details).length >= maxItems) {
      break;
    }

    if (detail.isTimestamp) {
      addFormattedTimestamp(details, detail.label, detail.value);
      continue;
    }

    addDetail(details, detail.label, detail.value);
  }
}

export function getProjectLabel(issue?: { project?: { name?: string; slug?: string } }) {
  return issue?.project?.name || issue?.project?.slug;
}
