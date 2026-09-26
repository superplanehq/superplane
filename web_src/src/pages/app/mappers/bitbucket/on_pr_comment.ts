import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { NodeMetadata, PullRequestCommentEvent } from "./types";
import { buildBitbucketSubtitle } from "./utils";

export interface OnPRCommentConfiguration {
  repository?: string;
  actions?: string[];
  contentFilter?: string;
}

const ACTION_LABELS: Record<string, string> = {
  comment_created: "Created",
  comment_updated: "Updated",
  comment_deleted: "Deleted",
};

function formatAction(action: string): string {
  return ACTION_LABELS[action] || action;
}

function commentTitle(event?: PullRequestCommentEvent): string {
  return event?.comment?.content?.raw?.trim() || "";
}

function commentSubtitle(event?: PullRequestCommentEvent): string {
  const pullRequest = event?.pullrequest;
  if (!pullRequest?.id) {
    return event?.actor?.display_name || "";
  }

  return `#${pullRequest.id} ${pullRequest.title || ""}`.trim();
}

/**
 * Renderer for the "bitbucket.onPRComment" trigger
 */
export const onPRCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as PullRequestCommentEvent;

    return {
      title: commentTitle(eventData),
      subtitle: buildBitbucketSubtitle(commentSubtitle(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PullRequestCommentEvent;

    return {
      Comment: commentTitle(eventData),
      Author: eventData?.actor?.display_name || "",
      "Pull Request": eventData?.pullrequest?.id ? `#${eventData.pullrequest.id}` : "",
      Title: eventData?.pullrequest?.title || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as NodeMetadata;
    const configuration = node.configuration as unknown as OnPRCommentConfiguration;
    const metadataItems = [];

    if (metadata?.repository) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.full_name || metadata.repository.name || "",
      });
    }

    if (configuration?.actions?.length) {
      metadataItems.push({
        icon: "activity",
        label: configuration.actions.map(formatAction).join(", "),
      });
    }

    if (configuration?.contentFilter) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.contentFilter,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: bitbucketIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as PullRequestCommentEvent;

      props.lastEventData = {
        title: commentTitle(eventData),
        subtitle: buildBitbucketSubtitle(commentSubtitle(eventData), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
