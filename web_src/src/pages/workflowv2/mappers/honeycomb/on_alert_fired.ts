import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import honeycombIcon from "@/assets/icons/integrations/honeycomb.svg";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";

interface OnAlertFiredConfiguration {
  datasetSlug?: string;
  trigger?: string;
}

interface OnAlertFiredEventData {
  name?: string;
  alert_type?: string;
  status?: string;
  summary?: string;
  trigger_url?: string;
  triggered_at?: string;
  severity?: string;
  result_value?: number;
}

export const onAlertFiredTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnAlertFiredEventData;

    return {
      title: buildEventTitle(eventData),
      subtitle: context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnAlertFiredEventData;

    return {
      Name: eventData?.name ?? "-",
      "Alert Type": eventData?.alert_type ?? "-",
      Status: eventData?.status ?? "-",
      Summary: eventData?.summary ?? "-",
      Severity: eventData?.severity ?? "-",
      "Result Value": eventData?.result_value?.toString() ?? "-",
      "Triggered At": eventData?.triggered_at ?? "-",
      "Trigger URL": eventData?.trigger_url ?? "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as unknown as OnAlertFiredConfiguration;
    const metadataItems = [];

    if (configuration?.datasetSlug) {
      metadataItems.push({
        icon: "database",
        label: configuration.datasetSlug,
      });
    }

    if (configuration?.trigger) {
      metadataItems.push({
        icon: "bell",
        label: configuration.trigger,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: honeycombIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnAlertFiredEventData;

      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "",
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildEventTitle(eventData?: OnAlertFiredEventData): string {
  const name = eventData?.name?.trim() || "Alert Fired";
  const alertType = eventData?.alert_type?.trim();

  if (!alertType) {
    return name;
  }

  return `${name} · ${alertType}`;
}
