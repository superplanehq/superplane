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

interface ListAlertsConfiguration {
  project?: string;
}

interface ListAlertsNodeMetadata {
  project?: {
    name?: string;
    slug?: string;
  };
}

interface SentryAlertRule {
  name?: string;
  projects?: string[];
  environment?: string | null;
  triggers?: Array<unknown>;
}

interface SentryListAlertsOutput {
  alerts?: SentryAlertRule[];
}

export const listAlertsMapper: ComponentBaseMapper = {
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
    const response = outputs?.default?.[0]?.data as SentryListAlertsOutput | undefined;
    const count = response?.alerts?.length ?? 0;
    const timestamp = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt));
    return [`${count} alert${count === 1 ? "" : "s"}`, timestamp].filter(Boolean).join(" · ");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const response = outputs?.default?.[0]?.data as SentryListAlertsOutput | undefined;
    const alerts = response?.alerts ?? [];
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Listed At", context.execution.createdAt);
    addOrderedDetails(details, [
      { label: "Alert Count", value: String(alerts.length) },
      { label: "Projects", value: summarizeProjects(alerts) },
      { label: "First Alert", value: alerts[0]?.name },
      { label: "First Alert Environment", value: normalizeNullable(alerts[0]?.environment) },
      { label: "Total Triggers", value: summarizeTriggerCount(alerts) },
    ]);

    return details;
  },
};

function buildMetadata(node: NodeInfo) {
  const configuration = node.configuration as ListAlertsConfiguration | undefined;
  const nodeMetadata = node.metadata as ListAlertsNodeMetadata | undefined;
  const metadata = [];

  const projectLabel = nodeMetadata?.project?.name || nodeMetadata?.project?.slug || configuration?.project;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  return metadata;
}

function summarizeProjects(alerts: SentryAlertRule[]) {
  const projects = [...new Set(alerts.flatMap((alert) => alert.projects || []).filter(Boolean))];
  return projects.slice(0, 3).join(", ") || undefined;
}

function summarizeTriggerCount(alerts: SentryAlertRule[]) {
  const totalTriggers = alerts.reduce((sum, alert) => sum + (alert.triggers?.length ?? 0), 0);
  return totalTriggers > 0 ? String(totalTriggers) : undefined;
}

function normalizeNullable(value: string | null | undefined) {
  return value || undefined;
}
