import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { formatTimeAgo } from "@/utils/date";
import {
  BaseNodeMetadata,
  CreateIncidentConfiguration,
  ServiceNowIncident,
  STATE_LABELS,
  URGENCY_LABELS,
  IMPACT_LABELS,
} from "./types";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { OutputPayload } from "../types";

export const createIncidentMapper: ComponentBaseMapper = {
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    return buildIncidentExecutionDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata;
  const configuration = node.configuration as CreateIncidentConfiguration;

  if (nodeMetadata?.instanceUrl) {
    const instanceName = nodeMetadata.instanceUrl.replace(/^https?:\/\//, "").replace(/\.service-now\.com$/, "");
    metadata.push({ icon: "globe", label: instanceName });
  }

  if (configuration.urgency) {
    const urgencyLabel = URGENCY_LABELS[configuration.urgency] || configuration.urgency;
    metadata.push({ icon: "funnel", label: `Urgency: ${urgencyLabel}` });
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

function getIncidentFromExecution(execution: CanvasesCanvasNodeExecution): ServiceNowIncident | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0].data?.result as ServiceNowIncident;
}

function buildIncidentExecutionDetails(execution: CanvasesCanvasNodeExecution): Record<string, any> {
  const details: Record<string, any> = {};

  if (execution.createdAt) {
    details["Executed at"] = new Date(execution.createdAt).toLocaleString();
  }

  const incident = getIncidentFromExecution(execution);
  if (incident) {
    if (incident.number) details["Number"] = incident.number;
    if (incident.sys_id) details["Sys ID"] = incident.sys_id;
    if (incident.short_description) details["Short Description"] = incident.short_description;
    if (incident.state) details["State"] = STATE_LABELS[incident.state] || incident.state;
    if (incident.urgency) details["Urgency"] = URGENCY_LABELS[incident.urgency] || incident.urgency;
    if (incident.impact) details["Impact"] = IMPACT_LABELS[incident.impact] || incident.impact;
    if (incident.sys_created_on) details["Created On"] = incident.sys_created_on;
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    details["Error"] = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  return details;
}
