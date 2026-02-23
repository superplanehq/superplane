import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { IncidentEvent } from "./types";

interface IncidentSummary {
  id?: string;
  title?: string;
  status?: string;
  severity?: string;
  services?: string[];
  teams?: string[];
}

interface OnEventEventData extends IncidentEvent {
  incident?: IncidentSummary;
  event_type?: string;
}

/**
 * Renderer for the "rootly.onIncidentTimelineEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incident = eventData?.incident;
    const title = eventData?.event || incident?.title || "Incident event";
    const contentParts = [incident?.title, incident?.severity, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    return getDetailsForIncidentEventPayload(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as {
      incidentStatus?: string[];
      severity?: string[];
      service?: string[];
      team?: string[];
      eventSource?: string[];
      visibility?: string;
    };
    const metadataItems = [];

    if (configuration?.incidentStatus?.length) {
      metadataItems.push({
        icon: "funnel",
        label: `Status: ${configuration.incidentStatus.join(", ")}`,
      });
    }

    if (configuration?.severity?.length) {
      metadataItems.push({
        icon: "alert-circle",
        label: `Severity: ${configuration.severity.join(", ")}`,
      });
    }

    if (configuration?.service?.length) {
      metadataItems.push({
        icon: "layers",
        label: `Service: ${configuration.service.join(", ")}`,
      });
    }

    if (configuration?.team?.length) {
      metadataItems.push({
        icon: "users",
        label: `Team: ${configuration.team.join(", ")}`,
      });
    }

    if (configuration?.eventSource?.length) {
      metadataItems.push({
        icon: "activity",
        label: `Source: ${configuration.eventSource.join(", ")}`,
      });
    }

    if (configuration?.visibility) {
      metadataItems.push({
        icon: "eye",
        label: `Visibility: ${configuration.visibility}`,
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
      const title = eventData?.event || incident?.title || "Incident event";
      const contentParts = [eventData?.kind, incident?.severity, incident?.status].filter(Boolean).join(" · ");
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

function getDetailsForIncidentEventPayload(eventData?: OnEventEventData): Record<string, string> {
  const details: Record<string, string> = {};

  if (!eventData) {
    return details;
  }

  if (eventData.created_at) {
    details["Created At"] = new Date(eventData.created_at).toLocaleString();
  }

  if (eventData.event) {
    details.Event = eventData.event;
  }

  if (eventData.id) {
    details["Event ID"] = eventData.id;
  }

  if (eventData.kind) {
    details.Kind = eventData.kind;
  }

  if (eventData.source) {
    details.Source = eventData.source;
  }

  if (eventData.visibility) {
    details.Visibility = eventData.visibility;
  }

  if (eventData.event_type) {
    details["Event Type"] = eventData.event_type;
  }

  if (eventData.occurred_at) {
    details["Occurred At"] = new Date(eventData.occurred_at).toLocaleString();
  }

  if (eventData.incident) {
    Object.assign(details, getDetailsForIncidentSummary(eventData.incident));
  }

  return details;
}

function getDetailsForIncidentSummary(incident: IncidentSummary): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = incident.id || "-";

  if (incident.title) {
    details.Title = incident.title;
  }

  if (incident.status) {
    details.Status = incident.status;
  }

  if (incident.severity) {
    details.Severity = incident.severity;
  }

  if (incident.services?.length) {
    details.Services = incident.services.join(", ");
  }

  if (incident.teams?.length) {
    details.Teams = incident.teams.join(", ");
  }

  return details;
}
