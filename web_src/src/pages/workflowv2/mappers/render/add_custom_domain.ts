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
import { stringOrDash } from "./common";
import { baseProps } from "./base";
import { defaultStateFunction } from "../stateRegistry";

interface AddCustomDomainConfiguration {
  service?: string;
  domainName?: string;
  waitForVerification?: boolean;
}

interface AddCustomDomainOutput {
  id?: string;
  name?: string;
  serviceId?: string;
  verificationStatus?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as AddCustomDomainConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }
  if (configuration?.domainName) {
    metadata.push({ icon: "globe", label: configuration.domainName });
  }
  if (configuration?.waitForVerification) {
    metadata.push({ icon: "shield-check", label: "Wait for verification" });
  }

  return metadata;
}

export const ADD_CUSTOM_DOMAIN_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const addCustomDomainStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[] } | undefined;
  if (outputs?.failed?.length) {
    return "failed";
  }

  return defaultStateFunction(execution);
};

export const ADD_CUSTOM_DOMAIN_STATE_REGISTRY: EventStateRegistry = {
  stateMap: ADD_CUSTOM_DOMAIN_STATE_MAP,
  getState: addCustomDomainStateFunction,
};

export const addCustomDomainMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;
    const result =
      (outputs?.success?.[0]?.data as AddCustomDomainOutput | undefined) ??
      (outputs?.failed?.[0]?.data as AddCustomDomainOutput | undefined);

    return {
      "Added At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Domain ID": stringOrDash(result?.id),
      "Domain Name": stringOrDash(result?.name),
      "Service ID": stringOrDash(result?.serviceId),
      "Verification Status": stringOrDash(result?.verificationStatus),
    };
  },
};
