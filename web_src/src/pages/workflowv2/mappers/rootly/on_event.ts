import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { IncidentEvent } from "./types";
import { getDetailsForIncidentEvent } from "./base";

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
  incident_event?: IncidentEvent & {
    kind?: string;
    user_display_name?: string;
    incident?: {
      id?: string;
      title?: string;
      status?: string;
      severity?: string;
    };
  };
  incident?: {
    id?: string;
    title?: string;
    status?: string;
    severity?: string;
  };
}

/**
 * Renderer for the "rootly.onEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incidentEvent = eventData?.incident_event;
    const incident = eventData?.incident;
    const title = incidentEvent?.event || incident?.title || "Timeline Event";
    const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const incidentEvent = eventData?.incident_event;
    const details: Record<string, string> = {};

    if (incidentEvent) {
      Object.assign(details, getDetailsForIncidentEvent(incidentEvent));

      if (incidentEvent.kind) {
        details["Kind"] = incidentEvent.kind;
      }

      if (incidentEvent.user_display_name) {
        details["User"] = incidentEvent.user_display_name;
      }
    }

    const incident = eventData?.incident;
    if (incident) {
      if (incident.title) {
        details["Incident"] = incident.title;
      }
      if (incident.status) {
        details["Incident Status"] = incident.status;
      }
      if (incident.severity) {
        details["Incident Severity"] = incident.severity;
      }
    }

    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[]; status?: string[]; severity?: string[]; visibility?: string[] };
    const metadataItems = [];

    if (configuration?.events) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${formattedEvents}`,
      });
    }

    if (configuration?.status?.length) {
      metadataItems.push({
        icon: "filter",
        label: `Status: ${configuration.status.join(", ")}`,
      });
    }

    if (configuration?.severity?.length) {
      metadataItems.push({
        icon: "filter",
        label: `Severity: ${configuration.severity.join(", ")}`,
      });
    }

    if (configuration?.visibility?.length) {
      metadataItems.push({
        icon: "filter",
        label: `Visibility: ${configuration.visibility.join(", ")}`,
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
      const incidentEvent = eventData?.incident_event;
      const incident = eventData?.incident;
      const title = incidentEvent?.event || incident?.title || "Timeline Event";
      const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title,
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
