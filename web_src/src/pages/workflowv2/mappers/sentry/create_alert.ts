import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addFormattedTimestamp, addOrderedDetails } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface AlertThresholdConfiguration {
  threshold?: number;
  notification?: {
    targetType?: string;
  };
}

interface CreateAlertConfiguration {
  project?: string;
  name?: string;
  environment?: string;
  critical?: AlertThresholdConfiguration;
  warning?: AlertThresholdConfiguration;
}

interface AlertRuleNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
}

interface SentryAlertRule {
  name?: string;
  environment?: string | null;
  projects?: string[];
  query?: string;
  aggregate?: string;
  triggers?: Array<{
    label?: string;
    alertThreshold?: number;
  }>;
}

export const createAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const alertRule = outputs?.default?.[0]?.data as SentryAlertRule | undefined;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [alertRule?.name, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const alertRule = outputs?.default?.[0]?.data as SentryAlertRule | undefined;
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
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
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

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as CreateAlertConfiguration | undefined;
  const nodeMetadata = node.metadata as AlertRuleNodeMetadata | undefined;
  const metadata = [];

  const projectLabel = getProjectLabel(nodeMetadata, configuration);
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }
  if (configuration?.name) {
    metadata.push({ icon: "siren", label: configuration.name });
  }
  const thresholdLabel = getThresholdLabel(configuration);
  if (thresholdLabel) {
    metadata.push({ icon: "triangle-alert", label: thresholdLabel });
  }

  return metadata.slice(0, 3);
}

function summarizeTriggers(alertRule: SentryAlertRule | undefined) {
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

function getProjectLabel(
  nodeMetadata: AlertRuleNodeMetadata | undefined,
  configuration: CreateAlertConfiguration | undefined,
) {
  return nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
}

function getThresholdLabel(configuration: CreateAlertConfiguration | undefined) {
  if (configuration?.critical?.threshold !== undefined) {
    return `Critical ≥ ${configuration.critical.threshold}`;
  }

  if (configuration?.warning?.threshold !== undefined) {
    return `Warning ≥ ${configuration.warning.threshold}`;
  }

  return undefined;
}
