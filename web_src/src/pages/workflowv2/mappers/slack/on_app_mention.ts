import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { TriggerProps } from "@/ui/trigger";
import slackIcon from "@/assets/icons/integrations/slack.svg";

interface OnAppMentionConfiguration {
  channel?: string;
}

interface OnAppMentionMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

interface AppMentionEventData {
  channel?: string;
  text?: string;
  user?: string;
  ts?: string;
  event_ts?: string;
  thread_ts?: string;
}

/**
 * Renderer for the "slack.onAppMention" trigger
 */
export const onAppMentionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as AppMentionEventData | undefined;
    const title = eventData?.text?.trim() ? eventData.text : "App mention";
    const subtitle = buildSubtitle(eventData?.user ? `Mention by ${eventData.user}` : "Mention", event.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = event.data?.data as AppMentionEventData | undefined;
    const mentionedAt = formatSlackTimestamp(eventData?.ts || eventData?.event_ts);

    return {
      "Mentioned At": mentionedAt || "",
      Channel: stringOrDash(eventData?.channel),
      User: stringOrDash(eventData?.user),
      Text: stringOrDash(eventData?.text),
      "Thread Timestamp": stringOrDash(eventData?.thread_ts),
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as OnAppMentionMetadata | undefined;
    const configuration = node.configuration as OnAppMentionConfiguration | undefined;
    const metadataItems = [];

    const channelLabel = metadata?.channel?.name || configuration?.channel;
    if (channelLabel) {
      metadataItems.push({
        icon: "hash",
        label: channelLabel,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as AppMentionEventData | undefined;
      const title = eventData?.text?.trim() ? eventData.text : "App mention";
      const subtitle = buildSubtitle(eventData?.user ? `Mention by ${eventData.user}` : "Mention", lastEvent.createdAt);

      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}

function formatSlackTimestamp(value?: unknown): string | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }

  const raw = String(value);
  const seconds = Number.parseFloat(raw);
  if (!Number.isNaN(seconds)) {
    return new Date(seconds * 1000).toLocaleString();
  }

  const asDate = new Date(raw);
  if (!Number.isNaN(asDate.getTime())) {
    return asDate.toLocaleString();
  }

  return raw;
}
