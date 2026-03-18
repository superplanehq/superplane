import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";

interface OnDocumentIndexedConfiguration {
  index?: string;
}

interface OnDocumentIndexedPayload {
  id?: string;
  index?: string;
  source?: Record<string, unknown> & {
    "@timestamp"?: string;
    message?: string;
  };
}

export const onDocumentIndexedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const payload = context.event?.data as OnDocumentIndexedPayload | undefined;
    const title = payload?.index ? `New document in ${payload.index}` : "New document indexed";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const payload = context.event?.data as OnDocumentIndexedPayload | undefined;
    const details: Record<string, string> = {};
    if (context.event?.createdAt) details["Triggered At"] = new Date(context.event.createdAt).toLocaleString();
    if (payload?.id) details["Document ID"] = String(payload.id);
    if (payload?.index) details["Index"] = String(payload.index);

    const preview = getSourcePreview(payload?.source);
    if (preview) {
      details[preview.label] = preview.value;
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const config = node.configuration as OnDocumentIndexedConfiguration | undefined;
    const metadata = buildMetadata(config);

    if (lastEvent) {
      const payload = lastEvent.data as OnDocumentIndexedPayload | undefined;
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

function buildMetadata(config: OnDocumentIndexedConfiguration | undefined): MetadataItem[] {
  const metadata: MetadataItem[] = [];

  if (config?.index) {
    metadata.push({ icon: "database", label: config.index });
  }

  return metadata;
}

function truncateMessage(message: string): string {
  return message.length > 60 ? `${message.slice(0, 57)}...` : message;
}

function getSourcePreview(source: OnDocumentIndexedPayload["source"]): { label: string; value: string } | null {
  if (!source) {
    return null;
  }

  if (typeof source.message === "string" && source.message.trim() !== "") {
    return { label: "Message", value: truncateMessage(source.message) };
  }

  for (const [key, value] of Object.entries(source)) {
    if (key === "@timestamp" || key === "message" || value == null) {
      continue;
    }

    if (typeof value === "string") {
      return { label: key, value: truncateMessage(value) };
    }

    if (typeof value === "number" || typeof value === "boolean") {
      return { label: key, value: String(value) };
    }
  }

  return null;
}
