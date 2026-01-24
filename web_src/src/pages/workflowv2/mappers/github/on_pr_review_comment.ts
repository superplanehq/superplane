import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
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
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnPullRequestReviewCommentEventData;
    const prNumber = eventData?.issue?.number || "";
    const fileName = eventData?.comment?.path || "";
    const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

    return {
      title: title,
      subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, event.createdAt),
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnPullRequestReviewCommentEventData;

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
      appName: "github",
      iconColor: getColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnPullRequestReviewCommentEventData;
      const prNumber = eventData?.issue?.number || "";
      const fileName = eventData?.comment?.path || "";
      const title = fileName ? `#${prNumber} - Comment on ${fileName}` : `#${prNumber} - Review Comment`;

      props.lastEventData = {
        title: title,
        subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
