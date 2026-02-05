import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Comment, Issue } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnPullRequestReviewCommentConfiguration {
  contentFilter?: string;
}

interface OnPullRequestReviewCommentEventData {
  action?: string;
  comment?: Comment;
  issue?: Issue;
}

/**
 * Renderer for the "github.onPullRequestReviewComment" trigger
 */
export const onPullRequestReviewCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPullRequestReviewCommentEventData;
    const prNumber = eventData?.issue?.number || "";
    const fileName = eventData?.comment?.path || "";
    const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

    return {
      title: title,
      subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPullRequestReviewCommentEventData;

    const rootValues: Record<string, string> = {
      Author: eventData?.comment?.user?.login || "",
      "Comment Body": eventData?.comment?.body || "",
      "Comment URL": eventData?.comment?.html_url || "",
      "PR Number": eventData?.issue?.number?.toString() || "",
      "PR Title": eventData?.issue?.title || "",
      "PR URL": eventData?.issue?.pull_request?.url || "",
    };

    if (eventData?.comment?.path) {
      rootValues["File Path"] = eventData.comment.path;
    }

    if (eventData?.comment?.line) {
      rootValues["Line Number"] = eventData.comment.line.toString();
    }

    return rootValues;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnPullRequestReviewCommentConfiguration;
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
      const eventData = lastEvent.data as OnPullRequestReviewCommentEventData;
      const prNumber = eventData?.issue?.number || "";
      const fileName = eventData?.comment?.path || "";
      const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

      props.lastEventData = {
        title: title,
        subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
