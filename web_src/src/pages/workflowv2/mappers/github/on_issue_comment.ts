import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Issue, Comment } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnIssueCommentConfiguration {
  contentFilter?: string;
}

interface OnIssueCommentEventData {
  action?: string;
  issue?: Issue;
  comment?: Comment;
}

/**
 * Renderer for the "github.onIssueComment" trigger
 */
export const onIssueCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event.data as OnIssueCommentEventData;

    return {
      title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
      subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, context.event.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event.data as OnIssueCommentEventData;

    return {
      Issue: `#${eventData?.issue?.number}`,
      "Issue Title": eventData?.issue?.title || "",
      "Comment Action": eventData?.action || "",
      By: eventData?.comment?.user?.login || "",
      "Comment Body": eventData?.comment?.body?.substring(0, 100) || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnIssueCommentConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.contentFilter) {
      metadataItems.push({
        icon: "funnel",
        label: `Filter: ${configuration.contentFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIssueCommentEventData;

      props.lastEventData = {
        title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
        subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
