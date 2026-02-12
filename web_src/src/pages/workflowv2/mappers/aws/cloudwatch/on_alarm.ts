import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import { formatTimeAgo } from "@/utils/date";
import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { Predicate, stringOrDash } from "../../utils";
import { CloudWatchAlarmEvent, CloudWatchAlarmTriggerConfiguration, CloudWatchAlarmTriggerMetadata } from "./types";

function formatAlarmFilterLabel(alarms?: Predicate[]): string | undefined {
  if (!alarms || alarms.length === 0) {
    return undefined;
  }

  if (alarms.length === 1 && alarms[0]?.type === "matches" && alarms[0]?.value === ".*") {
    return undefined;
  }

  if (alarms.length === 1) {
    return alarms[0]?.value || undefined;
  }

  const firstValue = alarms[0]?.value;
  if (!firstValue) {
    return `${alarms.length} filters`;
  }

  return `${firstValue} +${alarms.length - 1}`;
}

function buildMetadataItems(
  metadata?: CloudWatchAlarmTriggerMetadata,
  configuration?: CloudWatchAlarmTriggerConfiguration,
): MetadataItem[] {
  const items: MetadataItem[] = [];
  const region = metadata?.region || configuration?.region;
  if (region) {
    items.push({
      icon: "globe",
      label: region,
    });
  }

  const alarmFilterLabel = formatAlarmFilterLabel(configuration?.alarms);
  if (alarmFilterLabel) {
    items.push({
      icon: "bell",
      label: alarmFilterLabel,
    });
  }

  return items;
}

/**
 * Renderer for the "aws.cloudwatch.onAlarm" trigger
 */
export const onAlarmTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as CloudWatchAlarmEvent;
    const detail = eventData?.detail;
    const alarmName = detail?.alarmName;
    const state = detail?.state?.value;

    let title = "CloudWatch alarm";
    if (alarmName && state) {
      title = `${alarmName} (${state})`;
    } else if (alarmName) {
      title = alarmName;
    }

    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt || "")) : "";
    return { title, subtitle };
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
    const metadata = node.metadata as CloudWatchAlarmTriggerMetadata | undefined;
    const configuration = node.configuration as CloudWatchAlarmTriggerConfiguration | undefined;
    const metadataItems = buildMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsIcon,
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
