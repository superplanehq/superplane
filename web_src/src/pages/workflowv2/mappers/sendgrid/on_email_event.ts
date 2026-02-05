import { CanvasesCanvasEvent, ComponentsNode, TriggersTrigger } from "@/api-client";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";

const eventLabels: Record<string, string> = {
  processed: "Processed",
  delivered: "Delivered",
  deferred: "Deferred",
  bounce: "Bounced",
  dropped: "Dropped",
  open: "Opened",
  click: "Clicked",
  spamreport: "Spam Report",
  unsubscribe: "Unsubscribed",
  group_unsubscribe: "Group Unsubscribe",
  group_resubscribe: "Group Resubscribe",
};

function formatEventLabel(event: string): string {
  return eventLabels[event] || event;
}

interface OnEmailEventData {
  event?: string;
  email?: string;
  category?: string[] | string;
  sg_message_id?: string;
  timestamp?: string;
}

export const onEmailEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: CanvasesCanvasEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnEmailEventData;
    const eventType = eventData?.event ? formatEventLabel(eventData.event) : "Email Event";
    const subtitle = buildSubtitle(eventData?.email, event.createdAt);

    return {
      title: eventType,
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: CanvasesCanvasEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnEmailEventData;
    const category = Array.isArray(eventData?.category) ? eventData?.category.join(", ") : eventData?.category;
    return {
      "Received At": eventData?.timestamp ? new Date(Number(eventData.timestamp) * 1000).toLocaleString() : "-",
      Event: eventData?.event || "-",
      Email: eventData?.email || "-",
      Category: category || "-",
      "Message ID": eventData?.sg_message_id || "-",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: CanvasesCanvasEvent) => {
    const configuration = node.configuration as { eventTypes?: string[]; categoryFilter?: string };
    const metadataItems = [];

    if (configuration?.eventTypes?.length) {
      const formattedEvents = configuration.eventTypes.map(formatEventLabel).join(", ");
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${formattedEvents}`,
      });
    }

    if (configuration?.categoryFilter) {
      metadataItems.push({
        icon: "tag",
        label: `Category: ${configuration.categoryFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: sendgridIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnEmailEventData;
      const eventType = eventData?.event ? formatEventLabel(eventData.event) : "Email Event";
      const subtitle = buildSubtitle(eventData?.email, lastEvent.createdAt);

      props.lastEventData = {
        title: eventType,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};

function buildSubtitle(content: string | undefined, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}
