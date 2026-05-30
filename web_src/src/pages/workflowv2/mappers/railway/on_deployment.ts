import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type React from "react";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import railwayIcon from "@/assets/icons/integrations/railway.svg";

interface RailwayEventData {
  type?: string;
  projectId?: string;
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  status?: string;
  timestamp?: string;
}

interface OnDeploymentConfiguration {
  project?: string;
  eventTypes?: string[];
}

const eventLabels: Record<string, string> = {
  "Deployment.deployed": "Deployed",
  "Deployment.failed": "Failed",
  "Deployment.crashed": "Crashed",
  "Deployment.redeployed": "Redeployed",
  "Deployment.building": "Building",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Deployment Event";
  }
  return eventLabels[event] || event;
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return String(value);
}

function formatTimestamp(value?: string, fallback?: string): string {
  const timestamp = value || fallback;
  if (!timestamp) {
    return "-";
  }
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString();
}

export const onDeploymentTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const event = context.event?.data as RailwayEventData | undefined;
    const projectLabel = event?.projectId || "Project";
    const statusLabel = formatEventLabel(context.event?.type || event?.type);
    const title = `${projectLabel} · ${statusLabel}`;

    return {
      title,
      subtitle: context.event?.createdAt ? renderTimeAgo(new Date(context.event.createdAt)) : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = context.event?.data as RailwayEventData | undefined;
    return {
      "Received At": formatTimestamp(context.event?.createdAt),
      "Event Type": stringOrDash(context.event?.type || event?.type),
      "Project ID": stringOrDash(event?.projectId),
      "Environment ID": stringOrDash(event?.environmentId),
      "Service ID": stringOrDash(event?.serviceId),
      "Deployment ID": stringOrDash(event?.deploymentId),
      Status: stringOrDash(event?.status),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnDeploymentConfiguration | undefined;

    const metadata: TriggerProps["metadata"] = [];
    if (configuration?.project) {
      metadata.push({
        icon: "folder",
        label: `Project: ${configuration.project}`,
      });
    }

    if (configuration?.eventTypes && configuration.eventTypes.length > 0) {
      const formattedEvents = configuration.eventTypes.map(formatEventLabel);
      metadata.push({
        icon: "funnel",
        label:
          formattedEvents.length > 2
            ? `Events: ${formattedEvents.length} selected`
            : `Events: ${formattedEvents.join(", ")}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: railwayIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const event = lastEvent.data as RailwayEventData | undefined;
      const projectLabel = event?.projectId || "Project";
      const statusLabel = formatEventLabel(lastEvent.type || event?.type);
      props.lastEventData = {
        title: `${projectLabel} · ${statusLabel}`,
        subtitle: lastEvent.createdAt ? renderTimeAgo(new Date(lastEvent.createdAt)) : "",
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
