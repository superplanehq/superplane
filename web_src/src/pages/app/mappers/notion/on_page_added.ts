import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import notionIcon from "@/assets/icons/integrations/notion.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import type { NotionNodeMetadata, NotionPageEnvelope, OnPageAddedConfiguration } from "./types";

function pageTitle(envelope: NotionPageEnvelope | undefined): string {
  return envelope?.data?.title?.trim() || "";
}

export const onPageAddedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const envelope = context.event?.data as NotionPageEnvelope | undefined;

    return {
      title: pageTitle(envelope) || "Page",
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const envelope = (context.event?.data ?? {}) as NotionPageEnvelope;
    const page = envelope.data;

    return {
      "Received At": context.event?.createdAt ? formatTimestampInUserTimezone(context.event.createdAt) : "-",
      Page: stringOrDash(page?.id),
      Title: stringOrDash(page?.title),
      Content: stringOrDash(page?.content),
      URL: stringOrDash(page?.url),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as NotionNodeMetadata | undefined;
    const configuration = node.configuration as OnPageAddedConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    const databaseLabel = metadata?.database?.name || configuration?.database;
    if (databaseLabel) {
      metadataItems.push({ icon: "database", label: databaseLabel });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: notionIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onPageAddedTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
