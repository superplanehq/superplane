import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";

// Map event values to display labels (matching backend configuration)
const eventLabels: Record<string, string> = {
  "incident.created": "Created",
  "incident.updated": "Updated",
  "incident.mitigated": "Mitigated",
  "incident.resolved": "Resolved",
  "incident.cancelled": "Cancelled",
  "incident.deleted": "Deleted",
};

function formatEventLabel(event: string): string {
  return (
    eventLabels[event] ||
    event.replace("incident.", "").charAt(0).toUpperCase() + event.replace("incident.", "").slice(1)
  );
}

interface OnIncidentEventData {
  event?: string;
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onIncident" trigger type
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: incident?.title || "Incident",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident!);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[] };
    const metadataItems = [];

    if (configuration?.events) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${formattedEvents}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident",
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
