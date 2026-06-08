import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";
import gcpIcon from "@/assets/icons/integrations/gcp.compute.svg";

interface TPUNodeConfiguration {
  name?: string;
  node?: string;
  location?: string;
  acceleratorType?: string;
}

interface TPUNodeOutputData {
  name?: string;
  location?: string;
  state?: string;
  health?: string;
  acceleratorType?: string;
  runtimeVersion?: string;
  ipAddresses?: string[];
}

function lastSegment(value: string): string {
  const trimmed = value.trim();
  const idx = trimmed.lastIndexOf("/");
  return idx >= 0 ? trimmed.slice(idx + 1) : trimmed;
}

export const tpuMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "cpu",
      metadata: tpuMetadata(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as TPUNodeOutputData | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Completed At"] = new Date(payload.timestamp).toLocaleString();
    }
    if (!data) return details;

    if (data.name) details["Node"] = data.name;
    if (data.location) details["Location"] = data.location;
    if (data.state) details["State"] = data.state;
    if (data.health) details["Health"] = data.health;
    if (data.acceleratorType) details["Accelerator"] = data.acceleratorType;
    if (data.runtimeVersion) details["Runtime"] = data.runtimeVersion;
    if (Array.isArray(data.ipAddresses) && data.ipAddresses.length > 0) {
      details["IP Addresses"] = data.ipAddresses.join(", ");
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function tpuMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as TPUNodeConfiguration | undefined;

  const nodeName = config?.name || config?.node;
  if (nodeName) {
    metadata.push({ icon: "cpu", label: lastSegment(nodeName) });
  }
  if (config?.location) {
    metadata.push({ icon: "map-pin", label: config.location });
  }
  if (config?.acceleratorType) {
    metadata.push({ icon: "layers", label: config.acceleratorType });
  }

  return metadata;
}
