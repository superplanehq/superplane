import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";

// Map event values to display labels (matching backend configuration)
const eventLabels: Record<string, string> = {
  "incident_event.created": "Created",
  "incident_event.updated": "Updated",
};

function formatEventLabel(event: string): string {
  return (
    eventLabels[event] ||
    event.replace("incident_event.", "").charAt(0).toUpperCase() + event.replace("incident_event.", "").slice(1)
  );
}

interface OnEventEventData {
  event?: string;
  event_content?: string;
  kind?: string;
  visibility?: string;
  user_display_name?: string;
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incident = eventData?.incident;
    const contentParts = [eventData?.kind, eventData?.visibility].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: eventData?.event_content || incident?.title || "Timeline Event",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const details: Record<string, string> = {};

    if (eventData?.event_content) {
      details["Event"] = eventData.event_content;
    }

    if (eventData?.kind) {
      details["Kind"] = eventData.kind;
    }

    if (eventData?.visibility) {
      details["Visibility"] = eventData.visibility;
    }

    if (eventData?.user_display_name) {
      details["User"] = eventData.user_display_name;
    }

    if (eventData?.incident?.title) {
      details["Incident"] = eventData.incident.title;
    }

    if (eventData?.incident?.status) {
      details["Incident Status"] = eventData.incident.status;
    }

    if (eventData?.incident?.severity) {
      details["Severity"] = eventData.incident.severity;
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[]; visibility?: string; eventKind?: string };
    const metadataItems = [];

    if (configuration?.events) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${formattedEvents}`,
      });
    }

    if (configuration?.visibility) {
      metadataItems.push({
        icon: "eye",
        label: `Visibility: ${configuration.visibility}`,
      });
    }

    if (configuration?.eventKind) {
      metadataItems.push({
        icon: "tag",
        label: `Kind: ${configuration.eventKind}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEventEventData;
      const incident = eventData?.incident;
      const contentParts = [eventData?.kind, eventData?.visibility].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: eventData?.event_content || incident?.title || "Timeline Event",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}
