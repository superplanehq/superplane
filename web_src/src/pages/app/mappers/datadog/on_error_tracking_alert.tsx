import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { formatTimeAgo } from "@/lib/date";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import { addDetail, addFormattedTimestamp } from "./utils";

interface ErrorTrackingAlertEventData {
  title?: string;
  body?: string;
  link?: string;
  alert_id?: string;
  alert_transition?: string;
  tags?: string;
}

export const onErrorTrackingAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as ErrorTrackingAlertEventData;
    const title = eventData?.title?.trim() || "Error Tracking alert";
    const transition = eventData?.alert_transition?.trim();

    const subtitleParts = [
      transition,
      context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : undefined,
    ]
      .filter(Boolean)
      .map((value) => String(value));

    return {
      title,
      subtitle: subtitleParts.join(" · "),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as ErrorTrackingAlertEventData;
    const details: Record<string, string> = {};

    addFormattedTimestamp(details, "Triggered At", context.event?.createdAt);
    addDetail(details, "Title", eventData?.title);
    addDetail(details, "Body", eventData?.body);
    addDetail(details, "Link", eventData?.link);
    addDetail(details, "Monitor ID", eventData?.alert_id);
    addDetail(details, "Transition", eventData?.alert_transition);
    addDetail(details, "Tags", eventData?.tags);

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: datadogIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      const { title, subtitle } = onErrorTrackingAlertTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
