import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import { MonitorAlert } from "./types";

export const onMonitorAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as MonitorAlert;
    const contentParts = [eventData?.alert_transition, eventData?.priority].filter(Boolean).join(" - ");
    const subtitle = buildSubtitle(contentParts, event.createdAt);

    return {
      title: eventData?.monitor_name || eventData?.title || "Monitor Alert",
      subtitle,
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as MonitorAlert;
    return getDetailsForMonitorAlert(eventData);
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const configuration = node.configuration as any;
    const metadataItems = [];

    if (configuration.alertTransitions) {
      metadataItems.push({
        icon: "funnel",
        label: `Transitions: ${configuration.alertTransitions.join(", ")}`,
      });
    }

    if (configuration.tags) {
      metadataItems.push({
        icon: "tag",
        label: `Tags: ${configuration.tags}`,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: datadogIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as MonitorAlert;
      const contentParts = [eventData?.alert_transition, eventData?.priority].filter(Boolean).join(" - ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);

      props.lastEventData = {
        title: eventData?.monitor_name || eventData?.title || "Monitor Alert",
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
    return `${content} - ${timeAgo}`;
  }

  return content || timeAgo;
}

export function getDetailsForMonitorAlert(alert: MonitorAlert): Record<string, string> {
  const details: Record<string, string> = {};

  if (alert?.date) {
    details["Triggered At"] = new Date(alert.date * 1000).toLocaleString();
  }

  if (alert?.id) {
    details["Alert ID"] = alert.id;
  }

  if (alert?.monitor_id) {
    details["Monitor ID"] = String(alert.monitor_id);
  }

  if (alert?.monitor_name) {
    details["Monitor Name"] = alert.monitor_name;
  }

  if (alert?.title) {
    details["Title"] = alert.title;
  }

  if (alert?.alert_type) {
    details["Alert Type"] = alert.alert_type;
  }

  if (alert?.alert_transition) {
    details["Alert Transition"] = alert.alert_transition;
  }

  if (alert?.priority) {
    details["Priority"] = alert.priority;
  }

  if (alert?.hostname) {
    details["Hostname"] = alert.hostname;
  }

  if (alert?.tags && alert.tags.length > 0) {
    details["Tags"] = alert.tags.join(", ");
  }

  if (alert?.body) {
    details["Body"] = alert.body;
  }

  if (alert?.org) {
    details["Organization"] = alert.org.name || String(alert.org.id);
  }

  return details;
}
