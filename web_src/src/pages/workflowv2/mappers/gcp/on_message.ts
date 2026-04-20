import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";

export const onMessageTriggerRenderer: TriggerRenderer = {
  getEventState: (_context: TriggerEventContext) => "triggered",

  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    const subtitleParts: (string | React.ReactNode)[] = [];
    if (context.event?.createdAt) {
      subtitleParts.push(renderTimeAgo(new Date(context.event.createdAt)));
    }

    return subtitleParts.join(" · ");
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
    const configuration = node.configuration as
      | { topic?: string; topicId?: string; subscription?: string; subscriptionId?: string }
      | undefined;
    const topic = configuration?.topic || configuration?.topicId;
    const subscription = configuration?.subscription || configuration?.subscriptionId;
    const metadata = [];
    if (topic) {
      metadata.push({ icon: "message-square", label: topic });
    }
    if (subscription) {
      metadata.push({ icon: "radio", label: subscription });
    }
    const eventSubtitle = lastEvent ? onMessageTriggerRenderer.subtitle({ event: lastEvent }) : undefined;
    return {
      title: node.name || definition.label || "On Message",
      iconSrc: gcpPubSubIcon,
      iconSlug: definition.icon || "gcp",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata,
      ...(lastEvent && {
        lastEventData: {
          subtitle: eventSubtitle || renderTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};
