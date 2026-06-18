import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import cloudsmithIcon from "@/assets/icons/integrations/cloudsmith.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";
import type { OnComplianceCheckCompletedMetadata } from "./types";

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
    const eventData = context.event?.data as ComplianceCheckEvent | undefined;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Package: stringOrDash(eventData?.name),
      Version: stringOrDash(eventData?.version),
      Repository:
        eventData?.namespace && eventData?.repository
          ? `${eventData.namespace}/${eventData.repository}`
          : stringOrDash(eventData?.repository),
      License: stringOrDash(eventData?.license),
      "OSI Approved": yesNo(eventData?.osi_approved),
      Quarantined: yesNo(eventData?.is_quarantined),
      "Policy Violated": yesNo(eventData?.policy_violated),
      Status: stringOrDash(eventData?.status),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnComplianceCheckCompletedMetadata | undefined;
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
