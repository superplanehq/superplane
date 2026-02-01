import { ComponentsNode, TriggersTrigger, CanvasesCanvasEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";

interface OnIncidentResolvedEventData {
  event?: string;
  incident?: Incident;
}

export const onIncidentResolvedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: CanvasesCanvasEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentResolvedEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.severity?.name, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, event.createdAt);

    return {
      title: incident?.title || "Incident resolved",
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: CanvasesCanvasEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentResolvedEventData;
    const incident = eventData?.incident;
    if (!incident) return {};

    const details: Record<string, string> = {};
    details.ID = incident.id || "-";
    details.Title = incident.title || "-";
    details.Status = incident.status || "-";
    details.Slug = incident.slug || "-";
    details["Sequential ID"] = incident.sequential_id != null ? String(incident.sequential_id) : "-";
    details["Resolution Message"] = incident.resolution_message || "-";
    if (incident.resolved_at) {
      details["Resolved At"] = new Date(incident.resolved_at).toLocaleString();
    }
    if (incident.url) {
      details["Incident URL"] = incident.url;
    }
    return details;
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: CanvasesCanvasEvent) => {
    const configuration = node.configuration as {
      severityFilter?: string[];
      serviceFilter?: string[];
      teamFilter?: string[];
    };

    const metadataItems = [];
    if (configuration?.severityFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Severities: ${configuration.severityFilter.length}` });
    }
    if (configuration?.serviceFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Services: ${configuration.serviceFilter.length}` });
    }
    if (configuration?.teamFilter?.length) {
      metadataItems.push({ icon: "funnel", label: `Teams: ${configuration.teamFilter.length}` });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentResolvedEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity?.name, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident resolved",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
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
