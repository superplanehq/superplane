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

interface OrderCertificatePackConfiguration {
  zone?: string;
  hosts?: string[];
  certificateAuthority?: string;
  validationMethod?: string;
}

interface DeleteCertificatePackConfiguration {
  certificatePack?: string;
}

interface CertificatePackOutput {
  zoneId?: string;
  packId?: string;
  pack?: {
    id?: string;
    certificate_authority?: string;
    hosts?: string[];
    status?: string;
    type?: string;
    validation_method?: string;
  };
  deleted?: boolean;
}

export const orderCertificatePackMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: orderMetadataList(context.node),
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
    if (!result?.pack) return details;

    const pack = result.pack;
    if (pack.id) details["Pack ID"] = pack.id;
    if (pack.status) details["Status"] = pack.status;
    if (pack.certificate_authority) details["CA"] = pack.certificate_authority;
    if (pack.validation_method) details["Validation"] = pack.validation_method;
    if (pack.hosts?.length) details["Hosts"] = pack.hosts.join(", ");

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

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

    if (result.packId) details["Pack ID"] = result.packId;
    if (result.zoneId) details["Zone ID"] = result.zoneId;
    details["Deleted"] = result.deleted ? "Yes" : "No";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function orderMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as OrderCertificatePackConfiguration | undefined;

  if (config?.hosts?.length) {
    const label = config.hosts.length === 1 ? config.hosts[0] : `${config.hosts.length} hosts`;
    metadata.push({ icon: "shield-check", label });
  }

  if (config?.certificateAuthority) {
    metadata.push({ icon: "award", label: config.certificateAuthority.replace("_", " ") });
  }

  return metadata;
}

function deleteMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as DeleteCertificatePackConfiguration | undefined;

  const packValue = config?.certificatePack?.trim();
  if (packValue) {
    const packId = packValue.includes("/") ? packValue.split("/")[1] : packValue;
    metadata.push({ icon: "shield-off", label: packId });
  }

  return metadata;
}
