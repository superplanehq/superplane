import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import { formatRelativeTime } from "@/utils/timezone";
import { TriggerProps } from "@/ui/trigger";

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
    const subtitle = formatRelativeTime(event.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = event.data?.data as AppMentionEventData | undefined;

    return {
      Channel: stringOrDash(eventData?.channel),
      User: stringOrDash(eventData?.user),
      Text: stringOrDash(eventData?.text),
      Timestamp: stringOrDash(eventData?.ts || eventData?.event_ts),
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
      iconSlug: "slack",
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as AppMentionEventData | undefined;
      const title = eventData?.text?.trim() ? eventData.text : "App mention";
      const subtitle = formatRelativeTime(lastEvent.createdAt);

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
