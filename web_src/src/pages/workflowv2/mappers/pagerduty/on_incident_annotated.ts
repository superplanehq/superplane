import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { Agent, ResourceRef } from "./types";

interface OnIncidentAnnotatedMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

interface AnnotatedIncident {
  id?: string;
  incident_key?: string;
  number?: number;
  title?: string;
  urgency?: string;
  status?: string;
  html_url?: string;
  created_at?: string;
  service?: ResourceRef;
  escalation_policy?: ResourceRef;
  assignees?: Array<{ summary?: string; html_url?: string }>;
}

interface Annotation {
  content?: string;
}

interface OnIncidentAnnotatedEventData {
  agent?: Agent;
  incident?: AnnotatedIncident;
  annotation?: Annotation;
}

/**
 * Renderer for the "pagerduty.onIncidentAnnotated" trigger type
 */
export const onIncidentAnnotatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as OnIncidentAnnotatedEventData;
    const incident = eventData?.incident;
    const agent = eventData?.agent;
    const contentParts = [agent?.summary, "added note"].filter(Boolean).join(" ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);

    return {
      title: `${incident?.id || ""} - ${incident?.title || ""}`,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as OnIncidentAnnotatedEventData;
    return getDetailsForAnnotatedIncident(eventData?.incident, eventData?.agent, eventData?.annotation);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnIncidentAnnotatedMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "bell",
        label: metadata.service.name,
      });
    }

    if (configuration?.contentFilter) {
      metadataItems.push({
        icon: "funnel",
        label: `Filter: ${configuration.contentFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent?.data as OnIncidentAnnotatedEventData;
      const incident = eventData?.incident;
      const agent = eventData?.agent;
      const contentParts = [agent?.summary, "added note"].filter(Boolean).join(" ");
      const subtitle = buildSubtitle(contentParts, lastEvent?.createdAt);

      props.lastEventData = {
        title: `${incident?.id || ""} - ${incident?.title || ""}`,
        subtitle,
        receivedAt: new Date(lastEvent?.createdAt || ""),
        state: "triggered",
        eventId: lastEvent?.id,
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

/**
 * Get details specifically for annotated incidents.
 * This shows the note content prominently and only includes fields that are present.
 */
function getDetailsForAnnotatedIncident(
  incident?: AnnotatedIncident,
  agent?: Agent,
  annotation?: Annotation,
): Record<string, string> {
  const details: Record<string, string> = {};

  // Show annotation content first (most important for this event type)
  if (annotation?.content) {
    details["Note Content"] = annotation.content;
  }

  // Agent who added the note
  if (agent?.summary) {
    details["Added By"] = agent.summary;
  }

  // Incident details - only add if present
  if (incident?.id) {
    details["Incident ID"] = incident.id;
  }
  if (incident?.title) {
    details["Incident Title"] = incident.title;
  }
  if (incident?.status) {
    details["Status"] = incident.status;
  }
  if (incident?.urgency) {
    details["Urgency"] = incident.urgency;
  }
  if (incident?.html_url) {
    details["Incident URL"] = incident.html_url;
  }

  // Service info
  if (incident?.service?.summary) {
    details["Service"] = incident.service.summary;
  }

  // Assignees
  if (incident?.assignees && incident.assignees.length > 0) {
    const assigneeNames = incident.assignees.map((a) => a.summary).filter(Boolean);
    if (assigneeNames.length > 0) {
      details["Assignees"] = assigneeNames.join(", ");
    }
  }

  // Escalation policy
  if (incident?.escalation_policy?.summary) {
    details["Escalation Policy"] = incident.escalation_policy.summary;
  }

  // Created at
  if (incident?.created_at) {
    details["Incident Created"] = new Date(incident.created_at).toLocaleString();
  }

  return details;
}
