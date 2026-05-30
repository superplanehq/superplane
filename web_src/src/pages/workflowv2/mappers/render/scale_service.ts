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

interface ScaleServiceConfiguration {
  service?: string;
  numInstances?: number;
}

interface ScaleServiceOutput {
  serviceId?: string;
  numInstances?: number;
  status?: string;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ScaleServiceConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }

  if (configuration?.numInstances) {
    metadata.push({ icon: "gauge", label: `Instances: ${configuration.numInstances}` });
  }

  return metadata;
}

export const SCALE_SERVICE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  scaled: {
    icon: "check-circle",
    textColor: "text-green-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-green-500",
  },
};

export const scaleServiceStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (outputs?.default?.length) {
    const result = outputs.default[0]?.data as ScaleServiceOutput | undefined;
    if (result?.status === "accepted") {
      return "scaled";
    }
  }

  return defaultStateFunction(execution);
};

export const SCALE_SERVICE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: SCALE_SERVICE_STATE_MAP,
  getState: scaleServiceStateFunction,
};

export const scaleServiceMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as ScaleServiceOutput | undefined;

    return {
      "Requested At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      Instances: result?.numInstances === undefined ? "-" : String(result.numInstances),
      Status: stringOrDash(result?.status),
    };
  },
};
