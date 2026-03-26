import type { EventSection } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import type { ExecutionInfo, NodeInfo, StateFunction, TriggerRenderer } from "../types";

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

/** Shape of metric alert rule payloads used by Create/Update Alert mappers */
export type SentryAlertRule = {
  name?: string;
  environment?: string | null;
  projects?: string[];
  query?: string;
  aggregate?: string;
  triggers?: SentryAlertRuleTrigger[];
};

/** Node metadata persisted for alert-rule components (create/update/delete/get) */
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
        critical?: { threshold?: number };
        warning?: { threshold?: number };
      }
    | undefined,
): string | undefined {
  if (configuration?.critical?.threshold !== undefined) {
    return `Critical ≥ ${configuration.critical.threshold}`;
  }

  if (configuration?.warning?.threshold !== undefined) {
    return `Warning ≥ ${configuration.warning.threshold}`;
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

export function addOrderedDetails(details: Record<string, string>, entries: Array<{ label: string; value?: string }>) {
  entries.forEach(({ label, value }) => {
    if (!value) {
      return;
    }

    details[label] = value;
  });
}

/** Project label for Sentry issue payloads (nested project object) */
export function getProjectLabel(issue?: { project?: { name?: string; slug?: string } }) {
  return issue?.project?.name || issue?.project?.slug;
}
