import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { Agent } from "./types";

interface OnIncidentStatusUpdateMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

interface IncidentReference {
  id?: string;
  html_url?: string;
  summary?: string;
  type?: string;
}

interface StatusUpdate {
  id?: string;
  message?: string;
  subject?: string;
  incident?: IncidentReference;
}

interface OnIncidentStatusUpdateEventData {
  agent?: Agent;
  incident?: IncidentReference;
  status_update?: StatusUpdate;
}

/**
 * Renderer for the "pagerduty.onIncidentStatusUpdate" trigger type
 */
export const onIncidentStatusUpdateTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnIncidentStatusUpdateEventData;
    const incident = eventData?.incident;
    const statusUpdate = eventData?.status_update;
    const subtitle = buildSubtitle(statusUpdate?.message?.substring(0, 50) || "", event.createdAt);

    return {
      title: incident?.summary || incident?.id || "Status Update",
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnIncidentStatusUpdateEventData;
    const incident = eventData?.incident;
    const statusUpdate = eventData?.status_update;

    const values: Record<string, string> = {};

    if (incident?.id) {
      values["Incident ID"] = incident.id;
    }
    if (incident?.summary) {
      values["Incident Summary"] = incident.summary;
    }
    if (incident?.html_url) {
      values["Incident URL"] = incident.html_url;
    }
    if (statusUpdate?.message) {
      values["Status Update Message"] = statusUpdate.message;
    }
    if (lastEvent.createdAt) {
      values["Updated At"] = new Date(lastEvent.createdAt).toLocaleString();
    }
    if (eventData?.agent?.summary) {
      values["Agent"] = eventData.agent.summary;
    }

    return values;
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnIncidentStatusUpdateMetadata;
    const metadataItems = [];

    if (metadata?.service?.name) {
      metadataItems.push({
        icon: "bell",
        label: metadata.service.name,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      appName: "pagerduty",
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnIncidentStatusUpdateEventData;
      const incident = eventData?.incident;
      const statusUpdate = eventData?.status_update;
      const subtitle = buildSubtitle(statusUpdate?.message?.substring(0, 50) || "", lastEvent.createdAt);

      props.lastEventData = {
        title: incident?.summary || incident?.id || "Status Update",
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
