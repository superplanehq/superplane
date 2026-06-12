import { getBackgroundColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import type { TriggerProps } from "@/ui/trigger";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";

interface Configuration {
  region?: string;
  instance?: string;
  state?: string;
  alarm?: string;
}

interface NodeMetadata {
  region?: string;
  instanceId?: string;
  instanceName?: string;
}

interface AlarmState {
  value?: string;
  reason?: string;
  timestamp?: string;
}

interface AlarmDetail {
  alarmName?: string;
  state?: AlarmState;
  previousState?: AlarmState;
}

interface AlarmEvent {
  account?: string;
  region?: string;
  time?: string;
  detail?: AlarmDetail;
}

function buildMetadata(configuration?: Configuration, nodeMetadata?: NodeMetadata): MetadataItem[] {
  const items: MetadataItem[] = [];

  const instanceLabel = nodeMetadata?.instanceName || nodeMetadata?.instanceId || configuration?.instance;
  if (instanceLabel) {
    items.push({ icon: "server", label: instanceLabel });
  }

  if (configuration?.alarm) {
    items.push({ icon: "funnel", label: configuration.alarm });
  }

  if (configuration?.state) {
    items.push({ icon: "bell", label: configuration.state });
  }

  const region = configuration?.region || nodeMetadata?.region;
  if (region) {
    items.push({ icon: "globe", label: region });
  }

  return items.slice(0, 3);
}

export const onEc2AlarmTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as AlarmEvent;
    const detail = eventData?.detail;
    const alarmName = detail?.alarmName;
    const state = detail?.state?.value;
    const previousState = detail?.previousState?.value;

    let title = "EC2 alarm state change";
    if (alarmName && state && previousState) {
      title = `${alarmName} \u2014 ${previousState} \u2192 ${state}`;
    } else if (alarmName) {
      title = alarmName;
    }

    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as AlarmEvent;
    const detail = eventData?.detail;
    const triggeredAt = context.event?.createdAt ? new Date(context.event.createdAt).toLocaleString() : undefined;

    return {
      "Triggered At": stringOrDash(triggeredAt),
      Alarm: stringOrDash(detail?.alarmName),
      State: stringOrDash(detail?.state?.value),
      "Previous State": stringOrDash(detail?.previousState?.value),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as Configuration | undefined;
    const nodeMetadata = node.metadata as NodeMetadata | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsEc2Icon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadata(configuration, nodeMetadata),
    };

    if (lastEvent) {
      const { title, subtitle } = onEc2AlarmTriggerRenderer.getTitleAndSubtitle({
        event: lastEvent,
      });
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
