import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident, IncidentEvent } from "./types";
import { buildTimeAgoSubtitle, getDetailsForIncident, getDetailsForIncidentEvent } from "./base";

// Map incident status values to display labels (matching backend configuration)
const incidentStatusLabels: Record<string, string> = {
  in_triage: "In Triage",
  started: "Started",
  detected: "Detected",
  acknowledged: "Acknowledged",
  mitigated: "Mitigated",
  resolved: "Resolved",
  closed: "Closed",
  cancelled: "Cancelled",
};

function formatIncidentStatus(status: string): string {
  return incidentStatusLabels[status] || status;
}

interface OnEventConfiguration {
  incidentStatuses?: string[];
  severities?: string[];
  services?: string[];
  teams?: string[];
  visibility?: string;
}

interface OnEventEventData extends IncidentEvent {
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onEvent" trigger type
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnEventEventData;
    const incident = eventData?.incident;
    const contentParts = [];

    if (eventData?.kind) {
      contentParts.push(eventData.kind);
    }

    const subtitle = buildTimeAgoSubtitle(contentParts.filter(Boolean).join(" · "), context.event?.createdAt);

    return {
      title: incident?.title || "Incident Event",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnEventEventData;
    const details = getDetailsForIncidentEvent(eventData);
    if (eventData?.incident) {
      Object.assign(details, getDetailsForIncident(eventData.incident));
    }
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnEventConfiguration;
    const metadataItems: Array<{ icon: string; label: string }> = [];

    addMetadata(metadataItems, "Incident Status", configuration?.incidentStatuses);
    addMetadata(metadataItems, "Severity", configuration?.severities);
    addMetadata(metadataItems, "Service", configuration?.services);
    addMetadata(metadataItems, "Team", configuration?.teams);
    addMetadata(metadataItems, "Visibility", configuration?.visibility);

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEventEventData;
      const incident = eventData?.incident;
      const contentParts = [];

      if (eventData?.kind) {
        contentParts.push(eventData.kind);
      }

      const subtitle = buildTimeAgoSubtitle(contentParts.filter(Boolean).join(" · "), lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident Event",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function addMetadata(
  metadataItems: Array<{ icon: string; label: string }>,
  label: string,
  values?: string | string[],
): void {
  if (!values || (Array.isArray(values) && values.length === 0)) {
    return;
  }

  const list = Array.isArray(values) ? values : [values];

  const formatted = label === "Incident Status" ? list.map(formatIncidentStatus).join(", ") : list.join(", ");

  metadataItems.push({
    icon: "funnel",
    label: `${label}: ${formatted}`,
  });
}
