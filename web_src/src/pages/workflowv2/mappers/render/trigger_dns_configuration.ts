import type { ComponentBaseProps, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { serviceMetadataLabel, stringOrDash, type RenderServiceNodeMetadata } from "./common";
import { baseProps } from "./base";
import { defaultStateFunction } from "../stateRegistry";

interface TriggerDNSConfigurationConfiguration {
  service?: string;
  domainName?: string;
}

interface TriggerDNSConfigurationOutput {
  name?: string;
  serviceId?: string;
  status?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as TriggerDNSConfigurationConfiguration | undefined;
  const nodeMetadata = node.metadata as RenderServiceNodeMetadata | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${serviceMetadataLabel(nodeMetadata, configuration.service)}` });
  }
  if (configuration?.domainName) {
    metadata.push({ icon: "globe", label: configuration.domainName });
  }

  return metadata;
}

export const TRIGGER_DNS_CONFIGURATION_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  verificationRequested: {
    icon: "shield-check",
    textColor: "text-green-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-green-500",
  },
};

export const triggerDNSConfigurationStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (outputs?.default?.length) {
    const result = outputs.default[0]?.data as TriggerDNSConfigurationOutput | undefined;
    if (result?.status === "accepted") {
      return "verificationRequested";
    }
  }

  return defaultStateFunction(execution);
};

export const TRIGGER_DNS_CONFIGURATION_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TRIGGER_DNS_CONFIGURATION_STATE_MAP,
  getState: triggerDNSConfigurationStateFunction,
};

export const triggerDNSConfigurationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as TriggerDNSConfigurationOutput | undefined;

    return {
      "Requested At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Domain Name": stringOrDash(result?.name),
      "Service ID": stringOrDash(result?.serviceId),
      Status: stringOrDash(result?.status),
    };
  },
};
