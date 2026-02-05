import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
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

const allEventTypes = Object.keys(eventLabels);

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEmailEventData;
    const eventType = eventData?.event ? formatEventLabel(eventData.event) : "Email Event";
    const title = buildTitle(eventData?.email, eventType);
    const subtitle = buildSubtitle(context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEmailEventData;
    const category = Array.isArray(eventData?.category) ? eventData?.category.join(", ") : eventData?.category;
    return {
      "Received At": eventData?.timestamp
        ? new Date(Number(eventData.timestamp) * 1000).toLocaleString()
        : context.event?.createdAt
          ? new Date(context.event.createdAt).toLocaleString()
          : "-",
      Event: eventData?.event || "-",
      Email: eventData?.email || "-",
      Category: category || "-",
      "Message ID": eventData?.sg_message_id || "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { eventTypes?: string[]; categoryFilter?: string } | undefined;
    const metadataItems = [];

    if (shouldDisplayEventTypes(configuration?.eventTypes)) {
      const formattedEvents = configuration!.eventTypes!.map(formatEventLabel);
      const label =
        formattedEvents.length > 3
          ? `Events: ${formattedEvents.length} selected`
          : `Events: ${formattedEvents.join(", ")}`;
      metadataItems.push({
        icon: "funnel",
        label,
      });
    }

    if (shouldDisplayCategoryFilter(configuration?.categoryFilter)) {
      metadataItems.push({
        icon: "tag",
        label: `Category: ${configuration!.categoryFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: sendgridIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEmailEventData;
      const eventType = eventData?.event ? formatEventLabel(eventData.event) : "Email Event";
      const title = buildTitle(eventData?.email, eventType);
      const subtitle = buildSubtitle(lastEvent.createdAt);

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

function buildTitle(email: string | undefined, eventType: string): string {
  if (email) {
    return `${email} · ${eventType}`;
  }

  return `Email Event · ${eventType}`;
}

function buildSubtitle(createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  return timeAgo;
}

function shouldDisplayEventTypes(eventTypes?: string[]): boolean {
  if (!eventTypes || eventTypes.length === 0) {
    return false;
  }

  const normalizedEventTypes = eventTypes.map((value) => value.toLowerCase().trim());
  if (normalizedEventTypes.length !== allEventTypes.length) {
    return true;
  }

  const allSelected = allEventTypes.every((eventType) => normalizedEventTypes.includes(eventType));
  return !allSelected;
}

function shouldDisplayCategoryFilter(categoryFilter?: string): boolean {
  if (!categoryFilter) {
    return false;
  }

  const trimmed = categoryFilter.trim();
  return trimmed !== "" && trimmed !== "*";
}
