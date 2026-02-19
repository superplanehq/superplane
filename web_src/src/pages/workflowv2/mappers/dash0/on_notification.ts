import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { OnNotificationPayload } from "./types";

export const onNotificationTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnNotificationPayload;
    return {
      title: buildEventTitle(eventData),
      subtitle: buildEventSubtitle(eventData, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, any> => {
    const eventData = context.event?.data as OnNotificationPayload;
    const values: Record<string, string> = {};
    if (eventData?.type) values["Type"] = eventData.type;
    if (eventData?.checkName) values["Check Name"] = eventData.checkName;
    if (eventData?.severity) values["Severity"] = eventData.severity;
    if (eventData?.timestamp) values["Timestamp"] = eventData.timestamp;
    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dash0Icon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnNotificationPayload;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: buildEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildEventTitle(eventData: OnNotificationPayload): string {
  const type = eventData?.type || "Notification";
  if (eventData?.checkName) {
    return `${type} · ${eventData.checkName}`;
  }
  return type;
}

function buildEventSubtitle(eventData: OnNotificationPayload, createdAt?: string): string {
  const parts: string[] = [];

  if (eventData?.severity) {
    parts.push(eventData.severity);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}
