import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

interface CreateApplicationConfiguration {
  compartmentId?: string;
  displayName?: string;
  subnetIds?: string;
}

interface CreateApplicationOutputData {
  applicationId?: string;
  displayName?: string;
  lifecycleState?: string;
  compartmentId?: string;
  timeCreated?: string;
}

type CreateApplicationOutputPayload = OutputPayload & {
  data?: CreateApplicationOutputData;
};

function getOutputData(context: ExecutionDetailsContext): CreateApplicationOutputData | undefined {
  const outputs = context.execution.outputs as { default?: CreateApplicationOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as CreateApplicationOutputData;
}

export const createApplicationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return {
      ...baseMapper.props(context),
      metadata: createApplicationMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext) {
    return baseMapper.subtitle(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const data = getOutputData(context);
    if (!data) return details;

    if (data.displayName) {
      details["Display Name"] = data.displayName;
    }
    if (data.lifecycleState) {
      details["State"] = data.lifecycleState;
    }
    if (data.applicationId) {
      details["Application ID"] = data.applicationId;
    }

    return details;
  },
};

function createApplicationMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as CreateApplicationConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }

  return items;
}
