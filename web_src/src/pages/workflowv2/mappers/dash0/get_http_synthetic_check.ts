import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { GetHttpSyntheticCheckConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const getHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, unknown> | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};

    const metrics = responseData.metrics as Record<string, unknown> | undefined;
    const config = responseData.configuration as Record<string, unknown> | undefined;
    const metadata = config?.metadata as Record<string, unknown> | undefined;
    const configSpec = config?.spec as Record<string, unknown> | undefined;
    const plugin = configSpec?.plugin as Record<string, unknown> | undefined;
    const pluginSpec = plugin?.spec as Record<string, unknown> | undefined;
    const request = pluginSpec?.request as Record<string, unknown> | undefined;
    const assertions = pluginSpec?.assertions as Record<string, unknown> | undefined;
    const schedule = configSpec?.schedule as Record<string, unknown> | undefined;
    const notifications = configSpec?.notifications as Record<string, unknown> | undefined;

    if (payload?.timestamp) {
      details["Executed At"] = new Date(payload.timestamp).toLocaleString();
    }

    const display = configSpec?.display as Record<string, unknown> | undefined;
    const name = metadata?.name || display?.name;
    if (name) {
      details["Name"] = String(name);
    }

    if (request?.url) {
      const method = request.method ? String(request.method).toUpperCase() : "GET";
      details["Target"] = `${method} ${request.url}`;
    }

    const criticalAssertions = assertions?.criticalAssertions as Array<Record<string, unknown>> | undefined;
    if (criticalAssertions && criticalAssertions.length > 0) {
      const parts = criticalAssertions.map((a) => {
        const kind = String(a.kind || "").replace(/_/g, " ");
        const spec = a.spec as Record<string, unknown> | undefined;
        const operator = spec?.operator ? String(spec.operator) : "";
        const value = spec?.value ? String(spec.value) : "";
        return `${kind} ${operator} ${value}`.trim();
      });
      details["Expected"] = parts.join(", ");
    }

    if (schedule) {
      const parts: string[] = [];
      if (schedule.interval) parts.push(`every ${schedule.interval}`);
      if (schedule.locations) {
        parts.push(`from ${(schedule.locations as string[]).join(", ")}`);
      }
      if (schedule.strategy) parts.push(`(${String(schedule.strategy).replace(/_/g, " ")})`);
      details["Scheduling"] = parts.join(" ");
    }

    if (notifications) {
      const channels = notifications.channels as string[] | undefined;
      details["Notification Channels"] = channels && channels.length > 0 ? channels.join(", ") : "None";
    }

    if (metrics?.totalRuns24h != null) {
      details["Total Runs (24h)"] = String(metrics.totalRuns24h);
    }

    if (metrics?.criticalRuns24h != null) {
      details["Critical Runs (24h)"] = String(metrics.criticalRuns24h);
    }

    if (configSpec?.enabled != null) {
      details["Enabled"] = configSpec.enabled ? "Yes" : "No";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetHttpSyntheticCheckConfiguration;

  if (configuration?.checkId) {
    const idPreview =
      configuration.checkId.length > 24 ? configuration.checkId.substring(0, 24) + "…" : configuration.checkId;
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: configuration.dataset });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
