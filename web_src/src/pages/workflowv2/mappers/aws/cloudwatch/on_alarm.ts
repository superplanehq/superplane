import { TriggerProps } from "@/ui/trigger";
import { MetadataItem } from "@/ui/metadataList";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import { formatTimeAgo } from "@/utils/date";
import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { Predicate, formatPredicate, stringOrDash } from "../../utils";
import { CloudWatchAlarmEvent } from "./types";

interface Configuration {
  region?: string;
  state?: string;
  alarms?: Predicate[];
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

  if (configuration?.alarms && configuration.alarms?.length > 0) {
    items.push({
      icon: "funnel",
      label: configuration.alarms?.map(formatPredicate).join(", "),
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
    const previousState = detail?.previousState?.value;

    let title = "CloudWatch alarm";
    if (alarmName && state && previousState) {
      title = `${alarmName} - ${previousState} → ${state}`;
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
