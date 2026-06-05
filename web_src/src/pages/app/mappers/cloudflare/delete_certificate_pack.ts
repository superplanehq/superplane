import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./base";
import {
  certificatePackHostsLabel,
  certificatePackId,
  certificatePackZoneLabel,
  type CertificatePackOutput,
} from "./certificate_pack_helpers";

interface DeleteCertificatePackConfiguration {
  certificatePack?: string;
  certificatePackDisplayName?: string;
}

export const deleteCertificatePackMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: deleteMetadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CertificatePackOutput | undefined;
    if (!result) return details;

    const zoneLabel = certificatePackZoneLabel(result);
    if (zoneLabel) details["Zone"] = zoneLabel;

    const hostsLabel = certificatePackHostsLabel(result);
    if (hostsLabel) {
      details["Hosts"] = hostsLabel;
    } else {
      const packId = certificatePackId(result);
      if (packId) details["Pack ID"] = packId;
    }

    details["Deleted"] = result.deleted ? "Yes" : "No";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function deleteMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as DeleteCertificatePackConfiguration | undefined;

  const packValue = config?.certificatePack?.trim();
  if (packValue) {
    const named = config?.certificatePackDisplayName?.trim();
    if (named) {
      metadata.push({ icon: "shield-off", label: named });
    } else {
      const packId = packValue.includes("/") ? packValue.split("/")[1]! : packValue;
      metadata.push({ icon: "shield-off", label: packId });
    }
  }

  return metadata;
}
