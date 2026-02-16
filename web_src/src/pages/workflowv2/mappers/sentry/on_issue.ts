import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { ActionUser, Issue, OnIssueEventData } from "./types";

interface OnIssueEventDataExtended extends OnIssueEventData {
  event?: string;
  issue?: Issue;
  actionUser?: ActionUser;
}

/**
 * Renderer for "sentry.onIssue" trigger type
 */
export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as OnIssueEventDataExtended;
    const issue = eventData?.issue;
    const contentParts = [issue?.level, issue?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: `${issue?.shortId || ""} - ${issue?.title || ""}`,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnIssueEventDataExtended;
    const issue = eventData?.issue;
    return {
      Issue: issue?.shortId || "",
      Title: issue?.title || "",
      Status: issue?.status || "",
      Level: issue?.level || "",
      Project: issue?.project?.name || "",
      ID: issue?.id || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (configuration.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: sentryIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventDataExtended;
      const issue = eventData?.issue;
      const contentParts = [issue?.level, issue?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: `${issue?.shortId || ""} - ${issue?.title || ""}`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}
