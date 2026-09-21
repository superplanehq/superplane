import type { TriggerProps } from "@/ui/trigger";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import type { Predicate } from "../../eventDisplay";
import { formatPredicate, stringOrDash } from "../../eventDisplay";
import type { CloudWatchAlarmEvent } from "./types";

interface Configuration {
  region?: string;
  state?: string;
  alarms?: Predicate[];
}

function alarmTitle(detail?: CloudWatchAlarmEvent["detail"]): string {
  const alarmName = detail?.alarmName;
  const state = detail?.state?.value;
  const previousState = detail?.previousState?.value;
  if (alarmName && state && previousState) {
    return `${alarmName} - ${previousState} → ${state}`;
  }
  if (alarmName) {
    return alarmName;
  }
  return "CloudWatch alarm";
}

function createdAtSubtitle(createdAt?: string) {
  return createdAt ? renderTimeAgo(new Date(createdAt)) : "";
}

function buildMetadataItems(configuration?: Configuration): MetadataItem[] {
  const items: MetadataItem[] = [];
  const region = configuration?.region;
  if (region) {
    items.push({
      icon: "globe",
      label: region,
    });
  }

  if (configuration?.state) {
    items.push({
      icon: "bell",
      label: configuration.state,
    });
  }

  const alarms = configuration?.alarms;
  if (alarms && alarms.length > 0) {
    items.push({
      icon: "funnel",
      label: alarms.map(formatPredicate).join(", "),
    });
  }

  return items;
}

/**
 * Renderer for the "aws.cloudwatch.onAlarm" trigger
 */
export const onAlarmTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as CloudWatchAlarmEvent;
    return {
      title: alarmTitle(eventData?.detail),
      subtitle: createdAtSubtitle(context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as CloudWatchAlarmEvent;
    const detail = eventData?.detail;

    return {
      Alarm: stringOrDash(detail?.alarmName),
      State: stringOrDash(detail?.state?.value),
      "Previous State": stringOrDash(detail?.previousState?.value),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as Configuration | undefined;
    const metadataItems = buildMetadataItems(configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsCloudwatchIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onAlarmTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
