import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  addFormattedTimestamp,
  addOrderedDetails,
  AlertRuleNodeMetadata,
  buildEventSections,
  getAlertRuleProjectLabel,
  getAlertRuleSelectionLabel,
  getAlertThresholdMetadataLabel,
  summarizeTriggers,
  type SentryAlertRule,
  type SentryAlertThresholdConfiguration,
} from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface UpdateAlertConfiguration {
  alertId?: string;
  project?: string;
  critical?: SentryAlertThresholdConfiguration;
  warning?: SentryAlertThresholdConfiguration;
}

export const updateAlertMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution
        ? buildEventSections(context.nodes, lastExecution, componentName, getTriggerRenderer, getState)
        : undefined,
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

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as UpdateAlertConfiguration | undefined;
  const nodeMetadata = node.metadata as AlertRuleNodeMetadata | undefined;
  const metadata = [];

  const alertLabel = getAlertRuleSelectionLabel(nodeMetadata, configuration);
  if (alertLabel) {
    metadata.push({ icon: "siren", label: alertLabel });
  }

  const projectLabel = getAlertRuleProjectLabel(nodeMetadata, configuration);
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  const thresholdLabel = getAlertThresholdMetadataLabel(configuration);
  if (thresholdLabel) {
    metadata.push({ icon: "triangle-alert", label: thresholdLabel });
  }

  return metadata.slice(0, 3);
}
