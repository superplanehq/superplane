import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface CreateComputeInstanceConfiguration {
  compartmentId?: string;
  availabilityDomain?: string;
  displayName?: string;
  shape?: string;
  imageOs?: string;
  imageId?: string;
  subnetId?: string;
  sshPublicKey?: string;
  ocpus?: number;
  memoryInGBs?: number;
  bootVolumeSizeGB?: number;
  bootVolumeVpusPerGB?: string;
  attachBlockVolume?: boolean;
  blockVolumeId?: string;
  enableShieldedInstance?: boolean;
  enableConfidentialComputing?: boolean;
}

interface CreateComputeInstanceOutputData {
  displayName?: string;
  lifecycleState?: string;
  shape?: string;
  availabilityDomain?: string;
  compartmentId?: string;
  region?: string;
  timeCreated?: string;
  publicIp?: string;
}

type CreateComputeInstanceOutputPayload = OutputPayload & {
  data?: CreateComputeInstanceOutputData;
};

interface CreateComputeInstanceExecutionMetadata {
  startedAt?: string;
}

function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as CreateComputeInstanceExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function getOutputData(context: ExecutionDetailsContext): CreateComputeInstanceOutputData | undefined {
  const outputs = context.execution.outputs as { default?: CreateComputeInstanceOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as CreateComputeInstanceOutputData;
}

export const createComputeInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    return {
      ...props,
      metadata: createComputeInstanceMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const data = getOutputData(context);

    const executedAt = getExecutedAt(context);
    if (executedAt) {
      details["Executed At"] = executedAt;
    }

    if (!data) return details;

    if (data.displayName) {
      details["Display Name"] = data.displayName;
    }

    if (data.lifecycleState) {
      details["State"] = data.lifecycleState;
    }

    if (data.shape) {
      details["Shape"] = data.shape;
    }

    if (data.availabilityDomain) {
      details["Availability Domain"] = data.availabilityDomain;
    }

    if (data.region) {
      details["Region"] = data.region;
    }

    if (data.publicIp) {
      details["Public IP"] = data.publicIp;
    }

    return details;
  },
};

function createComputeInstanceMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as CreateComputeInstanceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }

  if (config?.shape) {
    items.push({ icon: "cpu", label: config.shape });
  }

  if (config?.availabilityDomain) {
    items.push({ icon: "map-pin", label: config.availabilityDomain });
  }

  return items;
}
