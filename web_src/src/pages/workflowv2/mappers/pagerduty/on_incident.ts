import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatRelativeTime } from "@/utils/timezone";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";

interface OnIncidentMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

interface OnIncidentEventData {
  agent?: {
    html_url?: string;
    summary?: string;
  };
  incident?: {
    id?: string;
    title?: string;
    urgency?: string;
    status?: string;
    html_url?: string;
    service?: {
      summary?: string;
    };
  };
}

/**
 * Renderer for the "pagerduty.onIncident" trigger type
 */
export const onIncidentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentEventData;
    const incident = eventData?.incident;

    return {
      title: `${incident?.id || ""} - ${incident?.title || ""}`,
      subtitle: `${incident?.urgency || ""} - ${incident?.status || ""} - ${formatRelativeTime(event.createdAt!)}`,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentEventData;
    const incident = eventData?.incident;

    let values: Record<string, string> = {
      Title: incident?.title || "",
      Status: incident?.status || "",
      Urgency: incident?.urgency || "",
      Service: incident?.service?.summary || "",
      URL: incident?.html_url || "",
    };

    if (eventData?.agent) {
      values["Agent"] = eventData.agent.summary || "";
      values["Agent URL"] = eventData.agent.html_url || "";
    }

    return values;
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
      iconBackground: "bg-green-500",
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentEventData;
      const incident = eventData?.incident;

      props.lastEventData = {
        title: `${incident?.id || ""} - ${incident?.title || ""}`,
        subtitle: `${incident?.urgency || ""} - ${incident?.status || ""} - ${formatRelativeTime(lastEvent.createdAt!)}`,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
