import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import { Incident } from "./types";
import { getDetailsForIncident } from "./base";

interface OnIncidentEventData {
  event?: string;
  incident?: Incident;
}

/**
 * Renderer for the "rootly.onIncident" trigger type
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentEventData;
    const incident = eventData?.incident;
    const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, event.createdAt);

    return {
      title: incident?.title || "Incident",
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentEventData;
    return getDetailsForIncident(eventData?.incident!);
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const configuration = node.configuration as { events?: string[] };
    const metadataItems = [];

    if (configuration?.events) {
      metadataItems.push({
        icon: "funnel",
        label: `Events: ${configuration.events.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: rootlyIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentEventData;
      const incident = eventData?.incident;
      const contentParts = [incident?.severity, incident?.status].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.title || "Incident",
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
