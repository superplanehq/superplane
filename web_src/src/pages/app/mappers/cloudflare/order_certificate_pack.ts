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

interface OrderCertificatePackConfiguration {
  zone?: string;
  hosts?: string[];
  certificateAuthority?: string;
  validationMethod?: string;
  validityDays?: string | number;
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
    if (!result) return details;

    const zoneLabel = certificatePackZoneLabel(result);
    if (zoneLabel) details["Zone"] = zoneLabel;

    const pack = result.pack;
    if (!pack) return details;

    const hostsLabel = certificatePackHostsLabel(result);
    if (hostsLabel) {
      details["Hosts"] = hostsLabel;
    } else {
      const packId = certificatePackId(result);
      if (packId) details["Pack ID"] = packId;
    }

    if (pack.status) details["Status"] = pack.status;
    if (pack.certificate_authority) details["CA"] = pack.certificate_authority;
    if (pack.validation_method) details["Validation"] = pack.validation_method;
    if (pack.validity_days) details["Validity"] = certificateValidityLabel(pack.validity_days);

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
    metadata.push({ icon: "award", label: config.certificateAuthority.replace(/_/g, " ") });
  }

  if (config?.validityDays && certificateAuthoritySupportsValidityDays(config.certificateAuthority)) {
    metadata.push({ icon: "calendar-clock", label: certificateValidityLabel(config.validityDays) });
  }

  return metadata;
}

function certificateAuthoritySupportsValidityDays(certificateAuthority: string | undefined): boolean {
  return certificateAuthority === "google" || certificateAuthority === "ssl_com";
}

function certificateValidityLabel(validityDays: string | number): string {
  switch (String(validityDays)) {
    case "14":
      return "2 weeks";
    case "30":
      return "1 month";
    case "90":
      return "3 months";
    default:
      return `${validityDays} days`;
  }
}
