import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type { TriggerProps } from "@/ui/trigger";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { getCloudflareTunnelName } from "./metadata";

interface TunnelHealthConfiguration {
  tunnel?: string;
  newStatus?: string[];
}

interface TunnelHealthEventData {
  alert_type?: string;
  new_status?: string;
  newStatus?: string;
  tunnel_id?: string;
  tunnelId?: string;
  tunnel_name?: string;
  tunnelName?: string;
  account_id?: string;
}

const TUNNEL_HEALTH_STATUS_LABELS: Record<string, string> = {
  TUNNEL_STATUS_TYPE_HEALTHY: "Healthy",
  TUNNEL_STATUS_TYPE_DEGRADED: "Degraded",
  TUNNEL_STATUS_TYPE_DOWN: "Down",
};

function firstTrimmed(values: (string | undefined)[]): string {
  for (const v of values) {
    const t = v?.trim();
    if (t) return t;
  }
  return "";
}

function tunnelHealthStatusDisplay(value: string | undefined): string {
  const v = value?.trim();
  if (!v) return "";
  return TUNNEL_HEALTH_STATUS_LABELS[v] ?? v;
}

function tunnelHealthStatusesListDisplay(values: string[] | undefined): string {
  if (!values?.length) return "";
  return values.map((s) => tunnelHealthStatusDisplay(s)).join(", ");
}

function parseTunnelHealthEventData(raw: unknown): TunnelHealthEventData | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return undefined;
  }
  const envelope = raw as Record<string, unknown>;
  const nested = envelope["data"];
  if (nested && typeof nested === "object" && !Array.isArray(nested)) {
    return nested as TunnelHealthEventData;
  }
  return envelope as TunnelHealthEventData;
}

function tunnelHealthNewStatusRawFromEvent(eventData?: TunnelHealthEventData): string {
  return firstTrimmed([eventData?.new_status, eventData?.newStatus]);
}

function tunnelLabelFromEventData(eventData?: TunnelHealthEventData): string {
  return firstTrimmed([eventData?.tunnel_name, eventData?.tunnelName, eventData?.tunnel_id, eventData?.tunnelId]);
}

function buildTunnelHealthRootEventValues(event: TriggerEventContext["event"]): Record<string, string> {
  const eventData = parseTunnelHealthEventData(event?.data);
  const statusRaw = tunnelHealthNewStatusRawFromEvent(eventData);
  const statusLabel = tunnelHealthStatusDisplay(statusRaw);

  return {
    "Triggered At": formatTriggeredAt(event?.createdAt),
    "Alert Type": displayValue(eventData?.alert_type),
    "New Status": displayValue(statusLabel),
    Tunnel: displayValue(tunnelLabelFromEventData(eventData)),
  };
}

function buildTunnelHealthLastEventData(
  lastEvent: NonNullable<TriggerRendererContext["lastEvent"]>,
): TriggerProps["lastEventData"] {
  const eventData = parseTunnelHealthEventData(lastEvent.data);
  return {
    title: buildEventTitle(eventData),
    subtitle: lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "",
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
}

export const onTunnelHealthTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = parseTunnelHealthEventData(context.event?.data);

    return {
      title: buildEventTitle(eventData),
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return buildTunnelHealthRootEventValues(context.event);
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as TunnelHealthConfiguration | undefined;
    const metadata = buildTunnelHealthTriggerMetadata(node, configuration, lastEvent);

    const props: TriggerProps = {
      title: node.name || definition.label || "Cloudflare tunnel health",
      iconSrc: cloudflareIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      props.lastEventData = buildTunnelHealthLastEventData(lastEvent);
    }

    return props;
  },
};

function buildTunnelHealthTriggerMetadata(
  node: TriggerRendererContext["node"],
  configuration: TunnelHealthConfiguration | undefined,
  lastEvent: TriggerRendererContext["lastEvent"],
): { icon: string; label: string }[] {
  const metadata: { icon: string; label: string }[] = [];

  const tunnelLabel = getCloudflareTunnelName(node.metadata) || getEventTunnelName(lastEvent) || configuration?.tunnel;
  if (tunnelLabel) {
    metadata.push({ icon: "server", label: tunnelLabel });
  }

  if (configuration?.newStatus?.length) {
    metadata.push({ icon: "activity", label: tunnelHealthStatusesListDisplay(configuration.newStatus) });
  }

  return metadata;
}

function buildEventTitle(eventData?: TunnelHealthEventData): string {
  const statusRaw = tunnelHealthNewStatusRawFromEvent(eventData);
  const status = tunnelHealthStatusDisplay(statusRaw) || "Status changed";
  const target = tunnelLabelFromEventData(eventData);
  return [target, status].filter(Boolean).join(" · ");
}

function getEventTunnelName(eventDataSource?: { data?: unknown }): string | undefined {
  const eventData = parseTunnelHealthEventData(eventDataSource?.data);
  const label = tunnelLabelFromEventData(eventData);
  return label || undefined;
}

function displayValue(value?: string): string {
  return value?.trim() || "-";
}

function formatTriggeredAt(createdAt?: string): string {
  if (!createdAt?.trim()) {
    return "-";
  }
  const parsed = new Date(createdAt);
  if (Number.isNaN(parsed.getTime())) {
    return "-";
  }
  return parsed.toLocaleString();
}
