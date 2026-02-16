import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { TriggerProps } from "@/ui/trigger";
import jiraIcon from "@/assets/icons/integrations/jira.svg";

interface OnIssueCreatedEventData {
  issue?: {
    key?: string;
    fields?: {
      summary?: string;
      creator?: {
        displayName?: string;
      };
    };
  };
}

export const onIssueCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIssueCreatedEventData | undefined;
    const issueKey = eventData?.issue?.key;
    const summary = eventData?.issue?.fields?.summary;
    const title = issueKey && summary ? `${issueKey}: ${summary}` : issueKey || summary || "Issue created";
    const creator = eventData?.issue?.fields?.creator?.displayName;
    const subtitle = buildSubtitle(creator ? `Created by ${creator}` : "Issue created", context.event?.createdAt);

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueCreatedEventData | undefined;

    return {
      "Issue Key": eventData?.issue?.key || "-",
      Summary: eventData?.issue?.fields?.summary || "-",
      Creator: eventData?.issue?.fields?.creator?.displayName || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: jiraIcon,
      iconSlug: "jira",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueCreatedEventData | undefined;
      const issueKey = eventData?.issue?.key;
      const summary = eventData?.issue?.fields?.summary;
      const title = issueKey && summary ? `${issueKey}: ${summary}` : issueKey || summary || "Issue created";
      const creator = eventData?.issue?.fields?.creator?.displayName;
      const subtitle = buildSubtitle(creator ? `Created by ${creator}` : "Issue created", lastEvent.createdAt);

      props.lastEventData = {
        title,
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
