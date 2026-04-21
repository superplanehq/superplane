import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import honeycombIcon from "@/assets/icons/integrations/honeycomb.svg";
import type { TriggerProps } from "@/pages/workflowv2/mappers/types";
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
  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    return context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";
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
      props.lastEventData = {
        subtitle: lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "",
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
