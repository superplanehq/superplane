import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface ManageInstancePowerConfiguration {
  instance?: string;
  instanceId?: string;
  action?: string;
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
  action?: string;
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

export const manageInstancePowerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as ManageInstancePowerConfiguration | undefined;
    const metadata: MetadataItem[] = [];
    const instance = config?.instance ?? config?.instanceId;

    if (instance) {
      metadata.push({ icon: "server", label: instance });
    }
    if (config?.action) {
      metadata.push({ icon: "zap", label: config.action });
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
    const config = context.node.configuration as ManageInstancePowerConfiguration | undefined;
    const data = getOutputData(context);
    return {
      "Executed At": executedAt(context),
      Action: config?.action ?? data?.action ?? "-",
      "Instance ID": data?.instanceId ?? "-",
      "Display Name": data?.displayName ?? "-",
      State: data?.lifecycleState ?? "-",
      Region: data?.region ?? "-",
    };
  },
};
