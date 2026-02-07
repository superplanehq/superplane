import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { OnAlertFiringEventData } from "./types";

/**
 * Renderer for the "grafana.onAlertFiring" trigger
 */
export const onAlertFiringTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;
    const alertName = getAlertName(eventData);
    const status = eventData?.status || "firing";
    const subtitle = buildSubtitle(status, context.event?.createdAt);

    return {
      title: alertName || "Grafana alert firing",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnAlertFiringEventData | undefined;

    return {
      Status: stringOrDash(eventData?.status || "firing"),
      "Alert Name": stringOrDash(getAlertName(eventData)),
      "Rule UID": stringOrDash(eventData?.ruleUid),
      "Rule ID": stringOrDash(eventData?.ruleId),
      "Org ID": stringOrDash(eventData?.orgId),
      "External URL": stringOrDash(eventData?.externalURL),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadataItems = [];

    if (lastEvent?.data) {
      const eventData = lastEvent.data as OnAlertFiringEventData;
      const alertName = getAlertName(eventData);
      if (alertName) {
        metadataItems.push({
          icon: "bell",
          label: alertName,
        });
      }
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: grafanaIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnAlertFiringEventData | undefined;
      const status = eventData?.status || "firing";
      const alertName = getAlertName(eventData);
      const subtitle = buildSubtitle(status, lastEvent.createdAt);

      props.lastEventData = {
        title: alertName || "Grafana alert firing",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function getAlertName(eventData?: OnAlertFiringEventData): string | undefined {
  if (!eventData) return undefined;

  if (eventData.title && eventData.title.trim() !== "") {
    return eventData.title;
  }

  const commonLabel = eventData.commonLabels?.alertname;
  if (commonLabel && commonLabel.trim() !== "") {
    return commonLabel;
  }

  const firstAlert = eventData.alerts?.[0];
  const labelName = firstAlert?.labels?.alertname;
  if (labelName && labelName.trim() !== "") {
    return labelName;
  }

  return undefined;
}

function buildSubtitle(status: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (status && timeAgo) {
    return `${status} - ${timeAgo}`;
  }

  return status || timeAgo;
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
