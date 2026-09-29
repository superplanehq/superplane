import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import honeycombIcon from "@/assets/icons/integrations/honeycomb.svg";
import type { TriggerProps } from "@/ui/trigger";
import { renderTimeAgo } from "@/components/TimeAgo";

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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as OnAlertFiredEventData;

    return {
      title: buildEventTitle(eventData),
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return alertFiredValues(context.event?.data as OnAlertFiredEventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as unknown as OnAlertFiredConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: honeycombIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: alertFiredMetadataItems(configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onAlertFiredTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function alertFiredValues(eventData?: OnAlertFiredEventData): Record<string, string> {
  return {
    Name: missingToDash(eventData?.name),
    "Alert Type": missingToDash(eventData?.alert_type),
    Status: missingToDash(eventData?.status),
    Summary: missingToDash(eventData?.summary),
    Severity: missingToDash(eventData?.severity),
    "Result Value": missingToDash(eventData?.result_value),
    "Triggered At": missingToDash(eventData?.triggered_at),
    "Trigger URL": missingToDash(eventData?.trigger_url),
  };
}

function missingToDash(value: string | number | undefined | null): string {
  if (value == null) {
    return "-";
  }

  return String(value);
}

function alertFiredMetadataItems(configuration?: OnAlertFiredConfiguration) {
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

  return metadataItems;
}

function buildEventTitle(eventData?: OnAlertFiredEventData): string {
  const name = eventData?.name?.trim() || "Alert Fired";
  const alertType = eventData?.alert_type?.trim();

  if (!alertType) {
    return name;
  }

  return `${name} · ${alertType}`;
}
