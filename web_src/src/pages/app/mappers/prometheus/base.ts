import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { GetAlertConfiguration, PrometheusAlertPayload } from "./types";

export const baseAlertMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildBaseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

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
  const [beforeTimestamps, afterTimestamps] = getOptionalAlertDetails(alert);
  const details: Record<string, string> = {
    "Alert Name": alert?.labels?.alertname || "-",
    State: alert?.status || "-",
    ...beforeTimestamps,
  };

  if (alert?.startsAt) {
    details["Starts At"] = new Date(alert.startsAt).toLocaleString();
  }

  if (alert?.endsAt && alert.endsAt !== "0001-01-01T00:00:00Z") {
    details["Ends At"] = new Date(alert.endsAt).toLocaleString();
  }

  return { ...details, ...afterTimestamps };
}

function getOptionalAlertDetails(alert: PrometheusAlertPayload): Record<string, string>[] {
  const labels = alert?.labels;
  const annotations = alert?.annotations;
  // Keep optional fields in their display order around the timestamps.
  const fieldGroups: Array<Array<[string, string | undefined]>> = [
    [
      ["Instance", labels?.instance],
      ["Job", labels?.job],
      ["Summary", annotations?.summary],
      ["Description", annotations?.description],
    ],
    [
      ["Value", alert?.value],
      ["Generator URL", alert?.generatorURL],
      ["Fingerprint", alert?.fingerprint],
      ["Alertmanager URL", alert?.externalURL],
    ],
  ];

  return fieldGroups.map((fields) => {
    const details: Record<string, string> = {};
    for (const [label, value] of fields) {
      if (value) {
        details[label] = value;
      }
    }
    return details;
  });
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
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
