import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type { TriggerProps } from "@/ui/trigger";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { getCloudflarePoolName } from "./metadata";

interface HealthAlertConfiguration {
  pool?: string;
  newHealth?: string[];
  eventSource?: string[];
}

interface HealthAlertEventData {
  alert_type?: string;
  event_source?: string;
  new_health?: string;
  pool_id?: string;
  pool_name?: string;
  origin_name?: string;
  load_balancer_name?: string;
}

export const onLoadBalancingHealthAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as HealthAlertEventData;

    return {
      title: buildEventTitle(eventData),
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as HealthAlertEventData;

    return {
      "Alert Type": displayValue(eventData?.alert_type),
      "Event Source": displayValue(eventData?.event_source),
      "New Health": displayValue(eventData?.new_health),
      Pool: displayValue(eventData?.pool_name || eventData?.pool_id),
      Origin: displayValue(eventData?.origin_name),
      "Load Balancer": displayValue(eventData?.load_balancer_name),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as HealthAlertConfiguration | undefined;
    const metadata = [];

    const poolLabel = getCloudflarePoolName(node.metadata) || getEventPoolName(lastEvent) || configuration?.pool;
    if (poolLabel) {
      metadata.push({ icon: "server", label: poolLabel });
    }

    if (configuration?.newHealth?.length) {
      metadata.push({ icon: "activity", label: configuration.newHealth.join(", ") });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Cloudflare health alert",
      iconSrc: cloudflareIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as HealthAlertEventData;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "",
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildEventTitle(eventData?: HealthAlertEventData): string {
  const health = eventData?.new_health?.trim() || "Health changed";
  const source = eventData?.event_source?.trim();
  const target =
    eventData?.origin_name?.trim() || eventData?.pool_name?.trim() || eventData?.load_balancer_name?.trim();

  return [target, source, health].filter(Boolean).join(" · ");
}

function getEventPoolName(eventDataSource?: { data?: unknown }): string | undefined {
  const eventData = eventDataSource?.data as HealthAlertEventData | undefined;
  return eventData?.pool_name?.trim() || undefined;
}

function displayValue(value?: string): string {
  return value?.trim() || "-";
}
