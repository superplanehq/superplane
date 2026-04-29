import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface UpdateInstanceConfiguration {
  instanceId?: string;
  displayName?: string;
}

interface InstanceOutputData {
  instanceId?: string;
  displayName?: string;
  lifecycleState?: string;
  shape?: string;
  availabilityDomain?: string;
  compartmentId?: string;
  region?: string;
  timeCreated?: string;
  publicIp?: string;
  privateIp?: string;
}

type InstanceOutputPayload = OutputPayload & {
  data?: InstanceOutputData;
};

function getOutputData(context: ExecutionDetailsContext): InstanceOutputData | undefined {
  const outputs = context.execution.outputs as { default?: InstanceOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as InstanceOutputData;
}

function executedAt(context: ExecutionDetailsContext): string {
  return context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
}

export const updateInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as UpdateInstanceConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (config?.instanceId) {
      metadata.push({ icon: "server", label: config.instanceId });
    }
    if (config?.displayName) {
      metadata.push({ icon: "tag", label: config.displayName });
    }

    return {
      ...props,
      metadata,
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = getOutputData(context);
    return {
      "Executed At": executedAt(context),
      "Instance ID": data?.instanceId ?? "-",
      "Display Name": data?.displayName ?? "-",
      State: data?.lifecycleState ?? "-",
      Shape: data?.shape ?? "-",
      Region: data?.region ?? "-",
    };
  },
};
