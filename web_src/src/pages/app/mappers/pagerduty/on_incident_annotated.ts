import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { TriggerProps } from "@/ui/trigger";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import type { Agent, ResourceRef } from "./types";

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = (context.event?.data as { data?: OnIncidentAnnotatedEventData } | undefined)?.data;
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
    const eventData = (context.event?.data as { data?: OnIncidentAnnotatedEventData } | undefined)?.data;
    return getDetailsForAnnotatedIncident(eventData?.incident, eventData?.agent, eventData?.annotation);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const props: TriggerProps = {
      title: node.name!,
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: annotatedIncidentMetadata(node),
    };

    if (lastEvent) {
      props.lastEventData = annotatedIncidentLastEvent(lastEvent);
    }

    return props;
  },
};

function buildSubtitle(content: string, createdAt?: string): string | React.ReactNode {
  if (content && createdAt) {
    return renderWithTimeAgo(content, new Date(createdAt));
  }

  if (createdAt) {
    return renderTimeAgo(new Date(createdAt));
  }

  return content;
}

function annotatedIncidentMetadata(node: TriggerRendererContext["node"]): MetadataItem[] {
  const metadata = node.metadata as unknown as OnIncidentAnnotatedMetadata;
  const configuration = node.configuration as { contentFilter?: string } | undefined;
  const metadataItems: MetadataItem[] = [];

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

  return metadataItems;
}

function annotatedIncidentLastEvent(
  lastEvent: NonNullable<TriggerRendererContext["lastEvent"]>,
): NonNullable<TriggerProps["lastEventData"]> {
  const eventData = lastEvent.data as OnIncidentAnnotatedEventData;
  const incident = eventData?.incident;
  const agent = eventData?.agent;
  const contentParts = [agent?.summary, "added note"].filter(Boolean).join(" ");

  return {
    title: `${incident?.id || ""} - ${incident?.title || ""}`,
    subtitle: buildSubtitle(contentParts, lastEvent.createdAt),
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
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

  assignAnnotatedNoteFields(details, annotation, agent);
  assignAnnotatedIncidentFields(details, incident);
  assignAnnotatedIncidentContext(details, incident);

  return details;
}

function assignAnnotatedNoteFields(details: Record<string, string>, annotation?: Annotation, agent?: Agent) {
  if (annotation?.content) {
    details["Note Content"] = annotation.content;
  }

  if (agent?.summary) {
    details["Added By"] = agent.summary;
  }
}

function assignAnnotatedIncidentFields(details: Record<string, string>, incident?: AnnotatedIncident) {
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
  if (incident?.service?.summary) {
    details["Service"] = incident.service.summary;
  }
}

function assignAnnotatedIncidentContext(details: Record<string, string>, incident?: AnnotatedIncident) {
  if (incident?.assignees && incident.assignees.length > 0) {
    const assigneeNames = incident.assignees.map((a) => a.summary).filter(Boolean);
    if (assigneeNames.length > 0) {
      details["Assignees"] = assigneeNames.join(", ");
    }
  }

  if (incident?.escalation_policy?.summary) {
    details["Escalation Policy"] = incident.escalation_policy.summary;
  }

  if (incident?.created_at) {
    details["Incident Created"] = new Date(incident.created_at).toLocaleString();
  }
}
