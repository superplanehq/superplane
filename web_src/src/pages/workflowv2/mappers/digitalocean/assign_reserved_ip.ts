import { ComponentBaseProps, EventSection, EventStateMap, DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { formatTimeAgo } from "@/utils/date";
import { AssignReservedIPConfiguration } from "./types";
import { defaultStateFunction } from "../stateRegistry";

export const reservedIPStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "digitalocean.reservedip.assign": {
    icon: "globe",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "ASSIGNED",
  },
  "digitalocean.reservedip.unassign": {
    icon: "globe",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "UNASSIGNED",
  },
};

export const ASSIGN_RESERVED_IP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: reservedIPStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const reservedIPEvent = outputs?.default?.find((output) => output.type?.startsWith("digitalocean.reservedip."));
    if (reservedIPEvent?.type && reservedIPStateMap[reservedIPEvent.type]) {
      return reservedIPEvent.type;
    }

    return "success";
  },
};

export const assignReservedIPMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: reservedIPStateMap,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const action = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    if (!action) return details;

    details["Action ID"] = action.id?.toString() || "-";
    details["Type"] = action.type || "-";
    details["Status"] = action.status || "-";

    if (action.resource_id) {
      details["Resource ID"] = action.resource_id?.toString() || "-";
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
  const configuration = node.configuration as AssignReservedIPConfiguration;

  if (configuration?.reservedIP) {
    metadata.push({ icon: "globe", label: `IP: ${configuration.reservedIP}` });
  }

  if (configuration?.action) {
    const actionLabels: Record<string, string> = {
      assign: "Assign",
      unassign: "Unassign",
    };
    const label = actionLabels[configuration.action] || configuration.action;
    metadata.push({ icon: "arrow-right-left", label: `Action: ${label}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode) {
    return [];
  }
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const reservedIPEvent = outputs?.default?.find((output) => output.type?.startsWith("digitalocean.reservedip."));

  const eventState =
    reservedIPEvent?.type && reservedIPStateMap[reservedIPEvent.type]
      ? reservedIPEvent.type
      : getState(componentName)(execution);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: eventState,
      eventId: execution.rootEvent!.id!,
    },
  ];
}
