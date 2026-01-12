import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Issue, Comment } from "./types";

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
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIssueCommentEventData;

    return {
      title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
      subtitle: `Comment by ${eventData?.comment?.user?.login || "unknown"}`,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIssueCommentEventData;

    return {
      Issue: `#${eventData?.issue?.number}`,
      "Issue Title": eventData?.issue?.title || "",
      "Comment Action": eventData?.action || "",
      "Comment By": eventData?.comment?.user?.login || "",
      "Comment Body": eventData?.comment?.body?.substring(0, 100) || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
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
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIssueCommentEventData;

      props.lastEventData = {
        title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
        subtitle: `Comment by ${eventData?.comment?.user?.login || "unknown"}`,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
