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
import gcpCloudDNSIcon from "@/assets/icons/integrations/gcp.clouddns.svg";

type CloudDNSChange = {
  id?: string | number;
  status?: string;
};

type CloudDNSRecord = {
  name?: string;
  type?: string;
};

type CloudDNSOutputData = {
  change?: CloudDNSChange;
  record?: CloudDNSRecord;
};

export const cloudDNSMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      iconSrc: gcpCloudDNSIcon,
      metadata: cloudDNSMetadata(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as CloudDNSOutputData | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Completed At"] = new Date(payload.timestamp).toLocaleString();
    }

    const change = data?.change;
    if (change?.id) {
      details["Change ID"] = String(change.id);
    }

    if (change?.status) {
      details["Status"] = String(change.status);
    }

    const record = data?.record;
    if (record?.name) {
      details["Record Name"] = String(record.name);
    }

    if (record?.type) {
      details["Record Type"] = String(record.type);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function cloudDNSMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as { managedZone?: string; name?: string; type?: string } | undefined;

  if (config?.managedZone) {
    metadata.push({ icon: "globe", label: String(config.managedZone) });
  }

  if (config?.name) {
    metadata.push({ icon: "tag", label: String(config.name) });
  }

  if (config?.type) {
    metadata.push({ icon: "layers", label: String(config.type) });
  }

  return metadata;
}
