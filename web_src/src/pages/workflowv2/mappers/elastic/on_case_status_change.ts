import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

interface OnCaseStatusChangeConfiguration {
  statuses?: string[];
}

export const onCaseStatusChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const payload = context.event?.data as Record<string, any> | undefined;
    const status = payload?.status ? ` to ${payload.status}` : "";
    const title = payload?.title ? `Case "${payload.title}" changed${status}` : "Case status changed";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = context.event?.data as Record<string, any> | undefined;
    const details: Record<string, string> = {};
    if (payload?.id) details["Case ID"] = String(payload.id);
    if (payload?.title) details["Title"] = String(payload.title);
    if (payload?.status) details["Status"] = String(payload.status);
    if (payload?.severity) details["Severity"] = String(payload.severity);
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as OnCaseStatusChangeConfiguration | undefined;
    const metadata: MetadataItem[] = [];
    if (config?.statuses && config.statuses.length > 0) {
      metadata.push({ icon: "filter", label: config.statuses.join(", ") });
    }

    if (lastEvent) {
      const payload = lastEvent.data as Record<string, any> | undefined;
      const status = payload?.status ? ` to ${payload.status}` : "";
      const title = payload?.title ? `Case "${payload.title}" changed${status}` : "Case status changed";
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
