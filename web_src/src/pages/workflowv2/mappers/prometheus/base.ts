import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { GetAlertConfiguration, PrometheusAlertPayload } from "./types";

export const baseAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildBaseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Retrieved At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const alert = outputs.default[0].data as PrometheusAlertPayload;
    return {
      ...details,
      ...getDetailsForAlert(alert),
    };
  },
};

export function buildBaseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: { name: string; label: string; color: string },
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

  return {
    iconSrc: prometheusIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name || componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? buildEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: getMetadata(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

export function getDetailsForAlert(alert: PrometheusAlertPayload): Record<string, string> {
  const details: Record<string, string> = {};

  details["Alert Name"] = alert?.labels?.alertname || "-";
  details["State"] = alert?.status || "-";

  if (alert?.labels?.instance) {
    details["Instance"] = alert.labels.instance;
  }

  if (alert?.labels?.job) {
    details["Job"] = alert.labels.job;
  }

  if (alert?.annotations?.summary) {
    details["Summary"] = alert.annotations.summary;
  }

  if (alert?.annotations?.description) {
    details["Description"] = alert.annotations.description;
  }

  if (alert?.startsAt) {
    details["Starts At"] = new Date(alert.startsAt).toLocaleString();
  }

  if (alert?.endsAt && alert.endsAt !== "0001-01-01T00:00:00Z") {
    details["Ends At"] = new Date(alert.endsAt).toLocaleString();
  }

  if (alert?.value) {
    details["Value"] = alert.value;
  }

  if (alert?.generatorURL) {
    details["Generator URL"] = alert.generatorURL;
  }

  if (alert?.fingerprint) {
    details["Fingerprint"] = alert.fingerprint;
  }

  if (alert?.externalURL) {
    details["Alertmanager URL"] = alert.externalURL;
  }

  return details;
}

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetAlertConfiguration | undefined;

  if (configuration?.alertName) {
    metadata.push({ icon: "bell", label: configuration.alertName });
  }

  if (configuration?.state && configuration.state !== "any") {
    metadata.push({ icon: "funnel", label: `State: ${configuration.state}` });
  }

  return metadata.slice(0, 3);
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
