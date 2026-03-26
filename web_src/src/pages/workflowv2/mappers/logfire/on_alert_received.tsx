import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import logfireIcon from "@/assets/icons/integrations/logfire.svg";
import type { MetadataItem } from "@/ui/metadataList";

interface LogfireAlertEventData {
  alertId?: string;
  alertName?: string;
  eventType?: string;
  severity?: string;
  message?: string;
  url?: string;
  timestamp?: string;
}

export const onAlertReceivedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as LogfireAlertEventData;
    const title = buildTitle(eventData);

    const subtitleParts: string[] = [];
    if (eventData?.eventType) {
      subtitleParts.push(eventData.eventType);
    }
    if (eventData?.severity) {
      subtitleParts.push(eventData.severity);
    }
    if (context.event?.createdAt) {
      subtitleParts.push(formatTimeAgo(new Date(context.event.createdAt)));
    }

    return {
      title,
      subtitle: subtitleParts.join(" · "),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as LogfireAlertEventData;

    const values: Record<string, string> = {
      "Alert Name": eventData?.alertName || "",
      Severity: eventData?.severity || "",
      Message: eventData?.message || "",
    };

    const matchingRows = extractMatchingRows(eventData?.message);
    if (matchingRows !== undefined) {
      values["Matching Rows"] = String(matchingRows);
    }

    values["Received At"] = getReceivedAtValue(context, eventData);

    if (eventData?.url) {
      values["View in Logfire"] = eventData.url;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;

    type LogfireOnAlertReceivedConfiguration = {
      projectId?: string;
      alertId?: string;
    };

    type LogfireOnAlertReceivedNodeMetadata = {
      project?: { id?: string; name?: string };
      alert?: { id?: string; name?: string };
    };

    const configuration = node.configuration as LogfireOnAlertReceivedConfiguration | undefined;
    const nodeMetadata = node.metadata as LogfireOnAlertReceivedNodeMetadata | undefined;

    const projectLabel = nodeMetadata?.project?.name?.trim() || configuration?.projectId?.trim();
    const alertLabel = nodeMetadata?.alert?.name?.trim() || configuration?.alertId?.trim();

    const metadata: MetadataItem[] = [];
    if (projectLabel) {
      metadata.push({ icon: "folder", label: `Project: ${projectLabel}` });
    }
    if (alertLabel) {
      metadata.push({ icon: "bell", label: `Alert: ${alertLabel}` });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: logfireIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadata.slice(0, 3),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as LogfireAlertEventData;
      const subtitleParts: string[] = [];
      if (eventData?.eventType) {
        subtitleParts.push(eventData.eventType);
      }
      if (eventData?.severity) {
        subtitleParts.push(eventData.severity);
      }
      if (lastEvent.createdAt) {
        subtitleParts.push(formatTimeAgo(new Date(lastEvent.createdAt)));
      }

      props.lastEventData = {
        title: buildTitle(eventData),
        subtitle: subtitleParts.join(" · "),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildTitle(eventData: LogfireAlertEventData | undefined): string {
  if (eventData?.alertName && eventData?.alertName.trim() !== "") {
    return eventData.alertName;
  }
  if (eventData?.message && eventData?.message.trim() !== "") {
    return eventData.message;
  }
  return "Logfire alert received";
}

function extractMatchingRows(message?: string): number | undefined {
  if (!message) return undefined;
  const match = message.match(/(\d+)\s+matching\s+rows?/i);
  return match ? Number.parseInt(match[1], 10) : undefined;
}

function getReceivedAtValue(context: TriggerEventContext, eventData: LogfireAlertEventData): string {
  const createdAt = context.event?.createdAt;
  if (!createdAt) return eventData.timestamp || "";

  const receivedAtDate = new Date(createdAt);
  if (Number.isNaN(receivedAtDate.getTime())) return eventData.timestamp || "";

  return receivedAtDate.toLocaleString();
}
