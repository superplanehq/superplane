import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";

interface OnTopicMessageMetadata {
  topic?: string;
}

interface OnTopicMessageEventData {
  topic?: string;
  messageId?: string;
  data?: unknown;
}

export const onTopicMessageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as OnTopicMessageEventData | undefined;
    const topic = data?.topic ?? "";
    const messageId = data?.messageId ?? "";
    const title = topic ? `Message on ${topic}` : "Pub/Sub message";
    const subtitle = messageId ? `ID: ${messageId}` : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return flattenObject(context.event?.data || {});
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnTopicMessageMetadata | undefined;

    const metadataItems = [];
    if (metadata?.topic) {
      metadataItems.push({ icon: "message-square", label: metadata.topic });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Pub/Sub • On Topic Message",
      iconSrc: gcpIcon,
      iconSlug: definition.icon || "cloud",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnTopicMessageEventData | undefined;
      const topic = eventData?.topic ?? "";

      props.lastEventData = {
        title: topic ? `Message on ${topic}` : "Pub/Sub message",
        subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
