import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";

export const onMessageTriggerRenderer: TriggerRenderer = {
  getEventState: (_context: TriggerEventContext) => "triggered",

  getTitleAndSubtitle: (_context: TriggerEventContext): { title: string; subtitle: string } => {
    return { title: "Pub/Sub message", subtitle: "" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as Record<string, any> | undefined;
    const details: Record<string, string> = {};
    if (data?.messageId) details["Message ID"] = String(data.messageId);
    if (data?.publishTime) details["Published At"] = new Date(data.publishTime as string).toLocaleString();
    if (context.event?.createdAt) details["Received At"] = new Date(context.event.createdAt).toLocaleString();
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    return {
      title: node.name || definition.label || "On Message",
      iconSrc: gcpPubSubIcon,
      iconSlug: definition.icon || "gcp",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: [],
      ...(lastEvent && {
        lastEventData: {
          title: "Pub/Sub message",
          subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
