import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Comment, Issue } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnPRCommentConfiguration {
  contentFilter?: string;
}

interface OnPRCommentEventData {
  action?: string;
  comment?: Comment;
  issue?: Issue;
}

/**
 * Renderer for the "github.onPRComment" trigger
 */
export const onPRCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPRCommentEventData;
    const prNumber = eventData?.issue?.number || "";
    const title = eventData?.issue?.title || "PR Comment";

    return {
      title: `#${prNumber} - ${title}`,
      subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPRCommentEventData;

    return {
      "Received At": context.event?.createdAt ? new Date(context.event.createdAt).toLocaleString() : "-",
      Author: eventData?.comment?.user?.login || "-",
      "Comment Body": eventData?.comment?.body || "-",
      "Comment URL": eventData?.comment?.html_url || "-",
      "PR Number": eventData?.issue?.number?.toString() || "-",
      "PR Title": eventData?.issue?.title || "-",
      "PR URL": eventData?.issue?.pull_request?.html_url || eventData?.issue?.pull_request?.url || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnPRCommentConfiguration;
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
      const eventData = lastEvent.data as OnPRCommentEventData;
      const prNumber = eventData?.issue?.number || "";
      const title = eventData?.issue?.title || "PR Comment";

      props.lastEventData = {
        title: `#${prNumber} - ${title}`,
        subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
