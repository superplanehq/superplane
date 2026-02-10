import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getDetailsForAlert } from "./base";
import { OnAlertConfiguration, PrometheusAlertPayload } from "./types";

const statusLabels: Record<string, string> = {
  firing: "Firing",
  resolved: "Resolved",
};

export const onAlertTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as PrometheusAlertPayload;
    const title = buildEventTitle(eventData);
    const subtitle = buildEventSubtitle(eventData, context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PrometheusAlertPayload;
    return getDetailsForAlert(eventData);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnAlertConfiguration | undefined;
    const metadata = [];

    if (configuration?.statuses && configuration.statuses.length > 0) {
      const formattedStatuses = configuration.statuses
        .map((status) => statusLabels[status] || status)
        .filter((status, index, values) => values.indexOf(status) === index);

      metadata.push({
        icon: "funnel",
        label: `Statuses: ${formattedStatuses.join(", ")}`,
      });
    }

    if (configuration?.alertNames && configuration.alertNames.length > 0) {
      const alertNames = configuration.alertNames.filter((value) => value.trim().length > 0);
      if (alertNames.length > 0) {
        metadata.push({
          icon: "bell",
          label:
            alertNames.length > 3
              ? `Alert Names: ${alertNames.length} selected`
              : `Alert Names: ${alertNames.join(", ")}`,
        });
      }
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: prometheusIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadata.slice(0, 3),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as PrometheusAlertPayload;
      props.lastEventData = {
        title: buildEventTitle(eventData),
        subtitle: buildEventSubtitle(eventData, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildEventTitle(eventData: PrometheusAlertPayload): string {
  const alertName = eventData?.labels?.alertname || "Alert";
  const sourceParts = [eventData?.labels?.instance, eventData?.labels?.job].filter(Boolean);

  if (sourceParts.length > 0) {
    return `${alertName} · ${sourceParts.join(" · ")}`;
  }

  return alertName;
}

function buildEventSubtitle(eventData: PrometheusAlertPayload, createdAt?: string): string {
  const parts: string[] = [];

  const status = eventData?.status;
  if (status) {
    parts.push(statusLabels[status] || status);
  }

  if (createdAt) {
    parts.push(formatTimeAgo(new Date(createdAt)));
  }

  return parts.join(" · ");
}
