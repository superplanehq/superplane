import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import type { CommitStatusEvent, NodeMetadata } from "./types";
import type { Predicate } from "../utils";
import { formatPredicate } from "../utils";
import { buildBitbucketSubtitle } from "./utils";

export interface OnCommitStatusConfiguration {
  repository?: string;
  states?: string[];
  keys?: Predicate[];
  refs?: Predicate[];
}

const STATE_LABELS: Record<string, string> = {
  INPROGRESS: "In progress",
  SUCCESSFUL: "Successful",
  FAILED: "Failed",
  STOPPED: "Stopped",
};

function formatState(state: string): string {
  return STATE_LABELS[state] || state;
}

function statusTitle(event?: CommitStatusEvent): string {
  const status = event?.commit_status;
  if (!status) {
    return "";
  }

  return `${status.name || status.key || ""} ${formatState(status.state || "")}`.trim();
}

function statusSubtitle(event?: CommitStatusEvent): string {
  return event?.commit_status?.refname || "";
}

function metadataItems(metadata: NodeMetadata, configuration: OnCommitStatusConfiguration): MetadataItem[] {
  const items: MetadataItem[] = [];
  const repository = metadata?.repository;

  if (repository) {
    items.push({ icon: "book", label: repository.full_name || repository.name || "" });
  }

  if (configuration?.states?.length) {
    items.push({ icon: "activity", label: configuration.states.map(formatState).join(", ") });
  }

  if (configuration?.keys?.length) {
    items.push({ icon: "hash", label: configuration.keys.map(formatPredicate).join(", ") });
  }

  if (configuration?.refs?.length) {
    items.push({ icon: "funnel", label: configuration.refs.map(formatPredicate).join(", ") });
  }

  return items;
}

/**
 * Renderer for the "bitbucket.onCommitStatus" trigger
 */
export const onCommitStatusTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as CommitStatusEvent;

    return {
      title: statusTitle(eventData),
      subtitle: buildBitbucketSubtitle(statusSubtitle(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as CommitStatusEvent;
    const status = eventData?.commit_status;

    return {
      Key: status?.key || "",
      Name: status?.name || "",
      State: status?.state || "",
      Ref: status?.refname || "",
      URL: status?.url || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as NodeMetadata;
    const configuration = node.configuration as unknown as OnCommitStatusConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: bitbucketIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as CommitStatusEvent;

      props.lastEventData = {
        title: statusTitle(eventData),
        subtitle: buildBitbucketSubtitle(statusSubtitle(eventData), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
