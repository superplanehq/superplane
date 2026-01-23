import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { Agent, Incident } from "./types";
import { getDetailsForIncident } from "./base";

interface OnIncidentMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

interface OnIncidentEventData {
  agent?: Agent;
  incident?: Incident;
}

/**
 * Renderer for the "pagerduty.onIncident" trigger type
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.urgency, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, event.createdAt);

    return {
      title: `${incident?.id || ""} - ${incident?.title || ""}`,
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident!, eventData.agent);
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnIncidentMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "bell",
        label: metadata.service.name,
      });
    }

    if (configuration.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    if (configuration.urgencies) {
      metadataItems.push({
        icon: "funnel",
        label: `Urgencies: ${configuration.urgencies.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.urgency, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: `${incident?.id || ""} - ${incident?.title || ""}`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
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
