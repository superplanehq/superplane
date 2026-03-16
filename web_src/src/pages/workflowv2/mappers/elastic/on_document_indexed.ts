import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

interface OnDocumentIndexedConfiguration {
  index?: string;
  timestampField?: string;
}

export const onDocumentIndexedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const payload = context.event?.data as Record<string, any> | undefined;
    const title = payload?.index ? `New document in ${payload.index}` : "New document indexed";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = context.event?.data as Record<string, any> | undefined;
    const details: Record<string, string> = {};
    if (payload?.id) details["Document ID"] = String(payload.id);
    if (payload?.index) details["Index"] = String(payload.index);
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as OnDocumentIndexedConfiguration | undefined;
    const metadata: MetadataItem[] = [];
    if (config?.index) {
      metadata.push({ icon: "database", label: config.index });
    }

    if (lastEvent) {
      const payload = lastEvent.data as Record<string, any> | undefined;
      const title = payload?.index ? `New document in ${payload.index}` : "New document indexed";
      return {
        title: node.name || definition.label || "Unnamed trigger",
        iconSrc: elasticIcon,
        collapsedBackground: getBackgroundColorClass(definition.color),
        metadata,
        lastEventData: {
          title,
          subtitle: formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      };
    }

    return {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: elasticIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };
  },
};
