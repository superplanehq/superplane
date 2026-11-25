import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { MetadataItem } from "@/ui/metadataList";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata, Comment, Issue } from "./types";
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
  getTitleAndSubtitle: (context: TriggerEventContext) => {
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
      ...prCommentFields(eventData?.comment),
      ...prFields(eventData?.issue),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: prCommentMetadata(node),
    };

    if (lastEvent) {
      props.lastEventData = prCommentLastEvent(lastEvent);
    }

    return props;
  },
};

function prCommentFields(comment: Comment | undefined): Record<string, string> {
  return {
    Author: comment?.user?.login || "-",
    "Comment Body": comment?.body || "-",
    "Comment URL": comment?.html_url || "-",
  };
}

function prFields(issue: Issue | undefined): Record<string, string> {
  return {
    "PR Number": issue?.number?.toString() || "-",
    "PR Title": issue?.title || "-",
    "PR URL": issue?.pull_request?.html_url || issue?.pull_request?.url || "-",
  };
}

function prCommentMetadata(node: TriggerRendererContext["node"]): MetadataItem[] {
  const metadata = node.metadata as unknown as BaseNodeMetadata;
  const configuration = node.configuration as unknown as OnPRCommentConfiguration;
  const metadataItems: MetadataItem[] = [];

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

  return metadataItems;
}

function prCommentLastEvent(
  lastEvent: NonNullable<TriggerRendererContext["lastEvent"]>,
): NonNullable<TriggerProps["lastEventData"]> {
  const eventData = lastEvent.data as OnPRCommentEventData;
  const prNumber = eventData?.issue?.number || "";
  const title = eventData?.issue?.title || "PR Comment";

  return {
    title: `#${prNumber} - ${title}`,
    subtitle: buildGithubSubtitle(`By ${eventData?.comment?.user?.login || "unknown"}`, lastEvent.createdAt),
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
}
