import type { ComponentBaseProps, EventSection, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import gcpIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { defaultStateFunction } from "../stateRegistry";
import { baseEventSections } from "./event_helpers";

interface VMInstanceNodeMetadata {
  instanceName?: string;
  zone?: string;
}

interface ManageVMInstancePowerConfiguration {
  instance?: string;
  operation?: string;
}

interface ManageVMInstancePowerOutputData {
  name?: string;
  zone?: string;
  status?: string;
  instanceId?: string;
  operation?: string;
}

const POWER_EVENT_PREFIX = "gcp.compute.vmInstance.power.";

export const powerStateMap: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  "gcp.compute.vmInstance.power.power_on": {
    icon: "power",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "STARTED",
  },
  "gcp.compute.vmInstance.power.power_off": {
    icon: "power-off",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "STOPPED",
  },
  "gcp.compute.vmInstance.power.reset": {
    icon: "rotate-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "RESET",
  },
  "gcp.compute.vmInstance.power.suspend": {
    icon: "pause",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "SUSPENDED",
  },
  "gcp.compute.vmInstance.power.resume": {
    icon: "play",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "RESUMED",
  },
};

const operationLabels: Record<string, string> = {
  power_on: "Start",
  power_off: "Stop",
  reset: "Reset",
  suspend: "Suspend",
  resume: "Resume",
};

export const MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: powerStateMap,
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;

    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const powerEvent = outputs?.default?.find((output) => output.type?.startsWith(POWER_EVENT_PREFIX));
    if (powerEvent?.type && powerStateMap[powerEvent.type]) {
      return powerEvent.type;
    }

    return "success";
  },
};

export const manageVMInstancePowerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "power",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Manage VM Power",
      eventSections: lastExecution ? powerEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: powerStateMap,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as ManageVMInstancePowerOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Instance Name"] = result.name;
    if (result.zone) details["Zone"] = result.zone;
    if (result.operation) details["Operation"] = operationLabels[result.operation] || result.operation;
    if (result.status) details["Status"] = result.status;

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as VMInstanceNodeMetadata | undefined;
  const configuration = node.configuration as ManageVMInstancePowerConfiguration | undefined;

  const instanceName = nodeMetadata?.instanceName || configuration?.instance;
  if (instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }
  if (nodeMetadata?.zone) {
    metadata.push({ icon: "map-pin", label: nodeMetadata.zone });
  }
  if (configuration?.operation) {
    const label = operationLabels[configuration.operation] || configuration.operation;
    metadata.push({ icon: "zap", label: `Operation: ${label}` });
  }

  return metadata;
}

// powerEventSections resolves the per-operation power state (when present) and
// delegates the rest of the section construction to the shared helper.
function powerEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const powerEvent = outputs?.default?.find((output) => output.type?.startsWith(POWER_EVENT_PREFIX));
  const override = powerEvent?.type && powerStateMap[powerEvent.type] ? powerEvent.type : undefined;

  return baseEventSections(nodes, execution, componentName, override);
}
