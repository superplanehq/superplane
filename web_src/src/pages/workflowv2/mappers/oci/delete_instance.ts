import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper, executedAt, getDefaultChannelOutputData } from "./base";

interface DeleteInstanceConfiguration {
  instance?: string;
  preserveBootVolume?: boolean;
}

interface InstanceOutputData {
  instanceId?: string;
  lifecycleState?: string;
}

export const deleteInstanceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const props = baseMapper.props(context);
    const config = context.node.configuration as DeleteInstanceConfiguration | undefined;
    const metadata: MetadataItem[] = [];

    if (config?.instance) {
      metadata.push({ icon: "trash-2", label: config.instance });
    }
    if (config?.preserveBootVolume) {
      metadata.push({ icon: "archive", label: "Preserve boot volume" });
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
    const config = context.node.configuration as DeleteInstanceConfiguration | undefined;
    const data = getDefaultChannelOutputData<InstanceOutputData>(context);
    return {
      "Executed At": executedAt(context),
      "Instance ID": data?.instanceId ?? "-",
      State: data?.lifecycleState ?? "-",
      "Preserve Boot Volume": config?.preserveBootVolume ? "Yes" : "No",
    };
  },
};
