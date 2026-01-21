import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Comment, PullRequest } from "./types";

interface OnPullRequestReviewCommentConfiguration {
  contentFilter?: string;
}

interface OnPullRequestReviewCommentEventData {
  action?: string;
  comment?: Comment;
  pull_request?: PullRequest;
}

/**
 * Renderer for the "github.onPullRequestReviewComment" trigger
 */
export const onPullRequestReviewCommentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnPullRequestReviewCommentEventData;

    const prNumber = eventData?.pull_request?.number || "";
    const fileName = eventData?.comment?.path || "";
    const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

    return {
      title: title,
      subtitle: eventData?.action || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnPullRequestReviewCommentEventData;

    const rootValues: Record<string, string> = {
      Action: eventData?.action || "",
      "PR Number": eventData?.pull_request?.number?.toString() || "",
      "PR Title": eventData?.pull_request?.title || "",
      "PR URL": eventData?.pull_request?._links?.html?.href || "",
      "Comment Body": eventData?.comment?.body || "",
      "Comment URL": eventData?.comment?.html_url || "",
      Author: eventData?.comment?.user?.login || "",
    };

    if (eventData?.comment?.path) {
      rootValues["File Path"] = eventData.comment.path;
    }

    if (eventData?.comment?.line) {
      rootValues["Line Number"] = eventData.comment.line.toString();
    }

    return rootValues;
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
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
      title: node.name!,
      iconSrc: githubIcon,
      iconColor: getColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnPullRequestReviewCommentEventData;
      const prNumber = eventData?.pull_request?.number || "";
      const fileName = eventData?.comment?.path || "";
      const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

      props.lastEventData = {
        title: title,
        subtitle: eventData?.action || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
