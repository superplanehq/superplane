import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type React from "react";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import vercelIcon from "@/assets/icons/integrations/vercel.svg";

export interface VercelEventData {
  eventType?: string;
  deploymentId?: string;
  name?: string;
  url?: string;
  readyState?: string;
  target?: string;
  projectId?: string;
}

interface OnEventConfiguration {
  eventTypes?: string[];
  project?: string;
}

/** Labels for event types as received in payloads (e.g. vercel.deployment.succeeded). */
const eventLabelsByType: Record<string, string> = {
  "vercel.deployment.created": "Deployment Created",
  "vercel.deployment.succeeded": "Deployment Succeeded",
  "vercel.deployment.error": "Deployment Failed",
  "vercel.deployment.canceled": "Deployment Canceled",
  "vercel.deployment.promoted": "Deployment Promoted",
};

/** Labels for event types as stored in configuration (e.g. deployment.succeeded). */
const eventLabelsByConfig: Record<string, string> = {
  "deployment.created": "Created",
  "deployment.succeeded": "Succeeded",
  "deployment.error": "Failed",
  "deployment.canceled": "Canceled",
  "deployment.promoted": "Promoted",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Vercel Event";
  }

  return eventLabelsByType[event] || eventLabelsByConfig[event] || event;
}

export const onDeploymentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const event = context.event?.data as VercelEventData | undefined;
    const projectLabel = event?.name || event?.projectId || "Vercel";
    const title =
      event && (event.name || event.projectId)
        ? `${projectLabel} · ${formatEventLabel(context.event?.type as string)}`
        : formatEventLabel(context.event?.type as string);

    return {
      title,
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = context.event?.data as VercelEventData | undefined;
    const values: Record<string, string> = {
      "Received At": formatTimestamp(context.event?.createdAt),
      Event: formatEventLabel(event?.eventType || (context.event?.type as string)),
      "Deployment ID": stringOrDash(event?.deploymentId),
      Project: stringOrDash(event?.name || event?.projectId),
      State: stringOrDash(event?.readyState),
      Target: stringOrDash(event?.target),
    };

    if (event?.url) {
      values["URL"] = event.url;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnEventConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: vercelIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadata(configuration),
    };

    if (lastEvent) {
      const event = lastEvent.data as VercelEventData;
      props.lastEventData = {
        title: `${event?.name || "Vercel"} · ${formatEventLabel(lastEvent.type as string)}`,
        subtitle: renderTimeAgo(new Date(lastEvent.createdAt)),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildMetadata(configuration: OnEventConfiguration | undefined): TriggerProps["metadata"] {
  const metadata: TriggerProps["metadata"] = [];

  const events = configuration?.eventTypes ?? [];
  if (events.length > 0) {
    const labels = events.map((event: string) => formatEventLabel(event));
    metadata.push({
      icon: "funnel",
      label: labels.length > 3 ? `Events: ${labels.length} selected` : `Events: ${labels.join(", ")}`,
    });
  }

  if (configuration?.project) {
    metadata.push({
      icon: "database",
      label: `Project: ${configuration.project}`,
    });
  }

  return metadata;
}

/** Shared: value or "-" for display. */
export function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return String(value);
}

/** Shared: format timestamp for display, or "-" if missing/invalid. */
export function formatTimestamp(value?: unknown): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value as string);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString();
}
