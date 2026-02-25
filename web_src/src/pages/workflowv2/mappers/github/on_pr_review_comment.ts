import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Comment, PullRequest } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnPRReviewCommentConfiguration {
  contentFilter?: string;
}

interface OnPRReviewCommentEventData {
  action?: string;
  comment?: Comment;
  pull_request?: PullRequest;
  review?: {
    body?: string;
    user?: {
      login?: string;
    };
  };
}

/**
 * Renderer for the "github.onPRReviewComment" trigger
 */
export const onPRReviewCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPRReviewCommentEventData;
    const prNumber = eventData?.pull_request?.number || "";
    const fileName = eventData?.comment?.path || "";
    const title = fileName
      ? `#${prNumber} - Review Comment on ${fileName}`
      : eventData?.review
        ? `#${prNumber} - PR Review`
        : `#${prNumber} - PR Review Comment`;

    const author = eventData?.comment?.user?.login || eventData?.review?.user?.login || "unknown";

    return {
      title,
      subtitle: buildGithubSubtitle(`By ${author}`, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPRReviewCommentEventData;

    const rootValues: Record<string, string> = {
      "Received At": context.event?.createdAt ? new Date(context.event.createdAt).toLocaleString() : "-",
      Author: eventData?.comment?.user?.login || eventData?.review?.user?.login || "-",
      "Comment Body": eventData?.comment?.body || eventData?.review?.body || "-",
      "Comment URL": eventData?.comment?.html_url || "-",
      "PR Number": eventData?.pull_request?.number?.toString() || "-",
      "PR Title": eventData?.pull_request?.title || "-",
      "PR URL":
        eventData?.pull_request?.html_url ||
        eventData?.pull_request?._links?.html?.href ||
        eventData?.pull_request?.url ||
        "-",
      "Branch Name": eventData?.pull_request?.head?.ref || "-",
      "Head SHA": eventData?.pull_request?.head?.sha || "-",
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
    const configuration = node.configuration as unknown as OnPRReviewCommentConfiguration;
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
      const eventData = lastEvent.data as OnPRReviewCommentEventData;
      const prNumber = eventData?.pull_request?.number || "";
      const fileName = eventData?.comment?.path || "";
      const title = fileName
        ? `#${prNumber} - Review Comment on ${fileName}`
        : eventData?.review
          ? `#${prNumber} - PR Review`
          : `#${prNumber} - PR Review Comment`;

      const author = eventData?.comment?.user?.login || eventData?.review?.user?.login || "unknown";

      props.lastEventData = {
        title,
        subtitle: buildGithubSubtitle(`By ${author}`, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
