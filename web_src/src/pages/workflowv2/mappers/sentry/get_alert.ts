import type { ComponentBaseProps } from "@/ui/componentBase";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { addFormattedTimestamp, addOrderedDetails, buildEventSections } from "./utils";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";

interface GetAlertConfiguration {
  project?: string;
  alertId?: string;
}

interface GetAlertNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
  alertName?: string;
}

interface SentryAlertRule {
  name?: string;
  aggregate?: string;
  query?: string;
  environment?: string | null;
  owner?: string;
  createdBy?: {
    name?: string;
    email?: string;
  };
  projects?: string[];
  triggers?: Array<{
    label?: string;
    actions?: Array<unknown>;
  }>;
}

export const getAlertMapper: ComponentBaseMapper = {
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

    addFormattedTimestamp(details, "Retrieved At", context.execution.createdAt);
    addOrderedDetails(details, [
      { label: "Name", value: alertRule?.name },
      { label: "Project", value: alertRule?.projects?.[0] },
      { label: "Created By", value: formatCreatedBy(alertRule) },
      { label: "Environment", value: normalizeNullable(alertRule?.environment) },
      { label: "Aggregate", value: alertRule?.aggregate },
      { label: "Owner", value: alertRule?.owner },
      { label: "Query", value: alertRule?.query },
      { label: "Trigger Labels", value: summarizeTriggers(alertRule) },
    ]);

    return details;
  },
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as GetAlertConfiguration | undefined;
  const nodeMetadata = node.metadata as GetAlertNodeMetadata | undefined;
  const metadata = [];

  const alertLabel = nodeMetadata?.alertName || configuration?.alertId;
  if (alertLabel) {
    metadata.push({ icon: "siren", label: alertLabel });
  }

  const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  return metadata;
}

function normalizeNullable(value: string | null | undefined) {
  return value || undefined;
}

function summarizeTriggers(alertRule: SentryAlertRule | undefined) {
  if (!alertRule?.triggers?.length) {
    return undefined;
  }

  return alertRule.triggers
    .map((trigger) => trigger.label)
    .filter(Boolean)
    .join(", ");
}

function formatCreatedBy(alertRule: SentryAlertRule | undefined) {
  return alertRule?.createdBy?.name || alertRule?.createdBy?.email;
}
