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
import { DropletNodeMetadata, ManageDropletPowerConfiguration } from "./types";
import { defaultStateFunction } from "../stateRegistry";

export const powerStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "digitalocean.droplet.power.power_on": {
    icon: "power",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "POWERED ON",
  },
  "digitalocean.droplet.power.power_off": {
    icon: "power-off",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "POWERED OFF",
  },
  "digitalocean.droplet.power.shutdown": {
    icon: "power-off",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "SHUTDOWN",
  },
  "digitalocean.droplet.power.reboot": {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "REBOOTED",
  },
  "digitalocean.droplet.power.power_cycle": {
    icon: "rotate-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "POWER CYCLED",
  },
};

export const MANAGE_DROPLET_POWER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: powerStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const powerEvent = outputs?.default?.find((output) => output.type?.startsWith("digitalocean.droplet.power."));
    if (powerEvent?.type && powerStateMap[powerEvent.type]) {
      return powerEvent.type;
    }

    return "success";
  },
};

export const manageDropletPowerMapper: ComponentBaseMapper = {
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
      eventStateMap: powerStateMap,
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
    details["Operation"] = action.type || "-";
    details["Status"] = action.status || "-";
    details["Droplet ID"] = action.resource_id?.toString() || "-";

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DropletNodeMetadata | undefined;
  const configuration = node.configuration as ManageDropletPowerConfiguration;

  if (nodeMetadata?.dropletName) {
    metadata.push({ icon: "hard-drive", label: nodeMetadata.dropletName });
  } else if (configuration?.dropletId) {
    metadata.push({ icon: "hard-drive", label: `Droplet: ${configuration.dropletId}` });
  }

  if (configuration?.operation) {
    const operationLabels: Record<string, string> = {
      power_on: "Power On",
      power_off: "Power Off",
      shutdown: "Shutdown",
      reboot: "Reboot",
      power_cycle: "Power Cycle",
    };
    const label = operationLabels[configuration.operation] || configuration.operation;
    metadata.push({ icon: "zap", label: `Operation: ${label}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  // Check if there's a custom power operation event in the outputs
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const powerEvent = outputs?.default?.find((output) => output.type?.startsWith("digitalocean.droplet.power."));

  // Use the custom power event type if available, otherwise fall back to default state
  const eventState =
    powerEvent?.type && powerStateMap[powerEvent.type] ? powerEvent.type : getState(componentName)(execution);

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
