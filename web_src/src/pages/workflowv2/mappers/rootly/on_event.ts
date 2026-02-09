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

interface OnEventEventData {
  event?: string;
  event_id?: string;
  issued_at?: string;
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incident = eventData?.incident;
    const eventType = eventData?.event ? formatEventLabel(eventData.event) : "";
    const contentParts = [eventType, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: incident?.title || "Event",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const details: Record<string, string> = {};

    if (eventData?.event) {
      details.Event = eventData.event;
    }

    if (eventData?.event_id) {
      details["Event ID"] = eventData.event_id;
    }

    if (eventData?.issued_at) {
      details["Issued At"] = new Date(eventData.issued_at).toLocaleString();
    }

    if (eventData?.incident) {
      const incidentDetails = getDetailsForIncident(eventData.incident);
      Object.assign(details, incidentDetails);
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as {
      events?: string[];
      status?: string;
      severity?: string;
      service?: string;
      team?: string;
      visibility?: string;
      kind?: string;
      source?: string;
    };
    const metadataItems = [];

    if (configuration?.events) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${formattedEvents}`,
      });
    }

    const filters = [];
    if (configuration?.status) filters.push(`Status: ${configuration.status}`);
    if (configuration?.severity) filters.push(`Severity: ${configuration.severity}`);
    if (configuration?.service) filters.push(`Service: ${configuration.service}`);
    if (configuration?.team) filters.push(`Team: ${configuration.team}`);
    if (configuration?.visibility) filters.push(`Visibility: ${configuration.visibility}`);
    if (configuration?.kind) filters.push(`Kind: ${configuration.kind}`);
    if (configuration?.source) filters.push(`Source: ${configuration.source}`);

    if (filters.length > 0) {
      metadataItems.push({
        icon: "filter",
        label: filters.join(", "),
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
      const eventType = eventData?.event ? formatEventLabel(eventData.event) : "";
      const contentParts = [eventType, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Event",
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
