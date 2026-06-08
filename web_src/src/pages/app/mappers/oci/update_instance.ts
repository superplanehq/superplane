import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper, executedAt, getDefaultChannelOutputData } from "./base";

interface UpdateInstanceConfiguration {
  instance?: string;
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

export const updateInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as UpdateInstanceConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (config?.instance) {
      metadata.push({ icon: "server", label: config.instance });
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
    const data = getDefaultChannelOutputData<InstanceOutputData>(context);
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
