import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { OnAlertEventData } from "./types";

const alertEventTypeLabels: Record<string, string> = {
  fired: "Fired",
  resolved: "Resolved",
};

function normalizeEventType(eventType: string | undefined): string {
  if (!eventType) {
    return "Alert";
  }

  const normalized = eventType.toLowerCase();
  return alertEventTypeLabels[normalized] || eventType;
}

function buildSubtitle(severity: string | undefined, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  const normalizedSeverity = severity ? severity.toUpperCase() : "";

  if (normalizedSeverity && timeAgo) {
    return `${normalizedSeverity} · ${timeAgo}`;
  }

  return normalizedSeverity || timeAgo;
}

export const onAlertEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnAlertEventData | undefined;
    const checkLabel = eventData?.checkName || eventData?.checkId || "Alert";
    const title = `${checkLabel} · ${normalizeEventType(eventData?.eventType)}`;
    const subtitle = buildSubtitle(eventData?.severity, context.event?.createdAt);
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnAlertEventData | undefined;
    return {
      "Event Type": eventData?.eventType || "-",
      "Check ID": eventData?.checkId || "-",
      "Check Name": eventData?.checkName || "-",
      Severity: eventData?.severity || "-",
      Summary: eventData?.summary || "-",
      Description: eventData?.description || "-",
      Timestamp: eventData?.timestamp ? new Date(eventData.timestamp).toLocaleString() : "-",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { eventTypes?: string[] } | undefined;
    const metadata = [];

    if (configuration?.eventTypes && configuration.eventTypes.length > 0) {
      metadata.push({
        icon: "funnel",
        label: `Events: ${configuration.eventTypes.map((eventType) => normalizeEventType(eventType)).join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "On Alert Event",
      iconSrc: dash0Icon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnAlertEventData | undefined;
      const checkLabel = eventData?.checkName || eventData?.checkId || "Alert";
      props.lastEventData = {
        title: `${checkLabel} · ${normalizeEventType(eventData?.eventType)}`,
        subtitle: buildSubtitle(eventData?.severity, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

