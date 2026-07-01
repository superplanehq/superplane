import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { WebhookTriggerNodeMetadata } from "./types";

interface SecurityScanEvent {
  event?: string;
  namespace?: string;
  repository?: string;
  name?: string;
  version?: string;
  slug_perm?: string;
  format?: string;
  security_scan_status?: string;
  vulnerability_scan_results_url?: string;
  has_vulnerabilities?: boolean;
  max_severity?: string;
  num_vulnerabilities?: number;
}

interface OnSecurityScanCompletedConfiguration {
  repository?: string;
}

/**
 * Renderer for the "cloudsmith.onSecurityScanCompleted" trigger.
 */
export const onSecurityScanCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const data = context.event?.data as SecurityScanEvent | undefined;
    const name = data?.name;
    const version = data?.version;

    const title = name ? `${name}${version ? ` ${version}` : ""}` : "Security scan";
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as SecurityScanEvent;
    const repository = data.namespace && data.repository ? `${data.namespace}/${data.repository}` : data.repository;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";
    const vulnerabilities = typeof data.num_vulnerabilities === "number" ? String(data.num_vulnerabilities) : "-";

    return {
      "Received At": receivedAt,
      Package: stringOrDash(data.name),
      Version: stringOrDash(data.version),
      Repository: stringOrDash(repository),
      "Security Scan": stringOrDash(data.security_scan_status),
      Vulnerabilities: vulnerabilities,
      "Max Severity": stringOrDash(data.max_severity),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as WebhookTriggerNodeMetadata | undefined;
    const configuration = node.configuration as OnSecurityScanCompletedConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const repositoryLabel = metadata?.repository
      ? `${metadata.repository.namespace}/${metadata.repository.slug}`
      : configuration?.repository;
    if (repositoryLabel) {
      metadataItems.push({ icon: "folder", label: repositoryLabel });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: cloudsmithIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onSecurityScanCompletedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
