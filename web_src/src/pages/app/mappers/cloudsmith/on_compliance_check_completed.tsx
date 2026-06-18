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

interface ComplianceCheckEvent {
  event?: string;
  namespace?: string;
  repository?: string;
  name?: string;
  version?: string;
  slug_perm?: string;
  license?: string;
  spdx_license?: string;
  osi_approved?: boolean;
  policy_violated?: boolean;
  is_quarantined?: boolean;
  status?: string;
  security_scan_status?: string;
  vulnerability_scan_results_url?: string;
  has_vulnerabilities?: boolean;
  max_severity?: string;
  num_vulnerabilities?: number;
}

interface OnComplianceCheckCompletedConfiguration {
  repository?: string;
}

const yesNo = (value: boolean | undefined): string => (value ? "Yes" : "No");

/**
 * Renderer for the "cloudsmith.onComplianceCheckCompleted" trigger.
 */
export const onComplianceCheckCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as ComplianceCheckEvent | undefined;
    const name = eventData?.name;
    const version = eventData?.version;

    const title = name ? `${name}${version ? ` ${version}` : ""}` : "Compliance check";
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = (context.event?.data ?? {}) as ComplianceCheckEvent;
    const repository = data.namespace && data.repository ? `${data.namespace}/${data.repository}` : data.repository;
    const receivedAt = context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-";
    const vulnerabilities = typeof data.num_vulnerabilities === "number" ? String(data.num_vulnerabilities) : "-";

    return {
      "Received At": receivedAt,
      Package: stringOrDash(data.name),
      Version: stringOrDash(data.version),
      Repository: stringOrDash(repository),
      License: stringOrDash(data.license),
      "OSI Approved": yesNo(data.osi_approved),
      Quarantined: yesNo(data.is_quarantined),
      "Policy Violated": yesNo(data.policy_violated),
      Status: stringOrDash(data.status),
      "Security Scan": stringOrDash(data.security_scan_status),
      Vulnerabilities: vulnerabilities,
      "Max Severity": stringOrDash(data.max_severity),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as WebhookTriggerNodeMetadata | undefined;
    const configuration = node.configuration as OnComplianceCheckCompletedConfiguration | undefined;
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
      const { title, subtitle } = onComplianceCheckCompletedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
