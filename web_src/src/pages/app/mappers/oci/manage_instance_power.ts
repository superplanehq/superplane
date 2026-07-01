import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper, executedAt, getDefaultChannelOutputData } from "./base";

interface ManageInstancePowerConfiguration {
  instance?: string;
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

export const manageInstancePowerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as ManageInstancePowerConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (config?.instance) {
      metadata.push({ icon: "server", label: config.instance });
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
    const data = getDefaultChannelOutputData<InstanceOutputData>(context);
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
