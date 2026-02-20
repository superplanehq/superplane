import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
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
import { MetadataItem } from "@/ui/metadataList";
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { formatTimeAgo } from "@/utils/date";
import {
  BaseNodeMetadata,
  IncidentRecord,
  STATE_LABELS,
  URGENCY_LABELS,
  IMPACT_LABELS,
} from "./types";
import { CanvasesCanvasNodeExecution } from "@/api-client";

function getIncidentFromExecution(execution: CanvasesCanvasNodeExecution): IncidentRecord | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }
  return outputs.default[0].data as IncidentRecord;
}

export const getIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "servicenow";

    return {
      iconSrc: snIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    const timeAgo = formatTimeAgo(new Date(context.execution.createdAt));
    const incident = getIncidentFromExecution(context.execution);
    if (incident?.number) {
      return `${incident.number} · ${timeAgo}`;
    }
    return timeAgo;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const nodeMetadata = context.node.metadata as BaseNodeMetadata;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Executed at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const incident = getIncidentFromExecution(context.execution);
    if (incident) {
      if (incident.number) details["Number"] = incident.number;
      if (incident.sys_id) {
        if (nodeMetadata?.instanceUrl) {
          details["Incident URL"] = `${nodeMetadata.instanceUrl}/incident.do?sys_id=${incident.sys_id}`;
        } else {
          details["Sys ID"] = incident.sys_id;
        }
      }
      if (incident.short_description) details["Short Description"] = incident.short_description;
      if (incident.state) details["State"] = STATE_LABELS[incident.state] || incident.state;
      if (incident.urgency) details["Urgency"] = URGENCY_LABELS[incident.urgency] || incident.urgency;
      if (incident.impact) details["Impact"] = IMPACT_LABELS[incident.impact] || incident.impact;
      if (incident.sys_created_on) details["Created On"] = incident.sys_created_on;
    }

    if (
      context.execution.resultMessage &&
      (context.execution.resultReason === "RESULT_REASON_ERROR" ||
        (context.execution.result === "RESULT_FAILED" &&
          context.execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
    ) {
      details["Error"] = {
        __type: "error",
        message: context.execution.resultMessage,
      };
    }

    return details;
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata | undefined;

  if (nodeMetadata?.instanceUrl) {
    const instanceName = nodeMetadata.instanceUrl.replace(/^https?:\/\//, "").replace(/\.service-now\.com$/, "");
    metadata.push({ icon: "globe", label: instanceName });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

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
