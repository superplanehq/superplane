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

export const onTunnelHealthTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = parseTunnelHealthEventData(context.event?.data);

    return {
      title: buildEventTitle(eventData),
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = parseTunnelHealthEventData(context.event?.data);
    const status =
      tunnelHealthStatusDisplay(eventData?.new_status?.trim()) ||
      tunnelHealthStatusDisplay(eventData?.newStatus?.trim());

    return {
      "Triggered At": formatTriggeredAt(context.event?.createdAt),
      "Alert Type": displayValue(eventData?.alert_type),
      "New Status": displayValue(status),
      Tunnel: displayValue(
        eventData?.tunnel_name || eventData?.tunnelName || eventData?.tunnel_id || eventData?.tunnelId,
      ),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as TunnelHealthConfiguration | undefined;
    const metadata = [];

    const tunnelLabel =
      getCloudflareTunnelName(node.metadata) || getEventTunnelName(lastEvent) || configuration?.tunnel;
    if (tunnelLabel) {
      metadata.push({ icon: "server", label: tunnelLabel });
    }

    if (configuration?.newStatus?.length) {
      metadata.push({ icon: "activity", label: tunnelHealthStatusesListDisplay(configuration.newStatus) });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Cloudflare tunnel health",
      iconSrc: cloudflareIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const eventData = parseTunnelHealthEventData(lastEvent.data);
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

function buildEventTitle(eventData?: TunnelHealthEventData): string {
  const statusRaw = eventData?.new_status?.trim() || eventData?.newStatus?.trim() || "";
  const status = tunnelHealthStatusDisplay(statusRaw) || "Status changed";
  const target =
    eventData?.tunnel_name?.trim() ||
    eventData?.tunnelName?.trim() ||
    eventData?.tunnel_id?.trim() ||
    eventData?.tunnelId?.trim();

  return [target, status].filter(Boolean).join(" · ");
}

function getEventTunnelName(eventDataSource?: { data?: unknown }): string | undefined {
  const eventData = parseTunnelHealthEventData(eventDataSource?.data);
  return (
    eventData?.tunnel_name?.trim() ||
    eventData?.tunnelName?.trim() ||
    eventData?.tunnel_id?.trim() ||
    eventData?.tunnelId?.trim() ||
    undefined
  );
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
