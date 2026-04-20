import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import type React from "react";
import type { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import octopusIcon from "@/assets/icons/integrations/octopus.svg";

interface OctopusEventData {
  eventType?: string;
  timestamp?: string;
  category?: string;
  message?: string;
  projectId?: string;
  projectName?: string;
  environmentId?: string;
  environmentName?: string;
  releaseId?: string;
  releaseName?: string;
  deploymentId?: string;
  serverUri?: string;
}

interface OnDeploymentEventConfiguration {
  eventCategories?: string[];
  project?: string;
  environment?: string;
}

interface OctopusNodeMetadata {
  projectName?: string;
  releaseName?: string;
  environmentName?: string;
}

/** Labels for event categories as received in payloads (dot-case). */
const eventLabelsByType: Record<string, string> = {
  "octopus.deployment.queued": "Deployment Queued",
  "octopus.deployment.started": "Deployment Started",
  "octopus.deployment.succeeded": "Deployment Succeeded",
  "octopus.deployment.failed": "Deployment Failed",
};

/** Labels for event categories as stored in configuration. */
const eventLabelsByConfig: Record<string, string> = {
  DeploymentQueued: "Deployment Queued",
  DeploymentStarted: "Deployment Started",
  DeploymentSucceeded: "Deployment Succeeded",
  DeploymentFailed: "Deployment Failed",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Octopus Event";
  }

  return eventLabelsByType[event] || eventLabelsByConfig[event] || event;
}

export const octopusTriggerRenderer: TriggerRenderer = {
  subtitle: (context: TriggerEventContext): string | React.ReactNode => {
    return buildSubtitle(context.event?.createdAt);
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = context.event?.data as OctopusEventData | undefined;
    const values: Record<string, string> = {
      "Received At": formatTimestamp(context.event?.createdAt),
      Event: stringOrDash(context.event?.type),
      "Event Type": stringOrDash(event?.eventType),
    };

    if (event?.projectId) {
      values["Project"] = event.projectName || event.projectId;
    }
    if (event?.environmentId) {
      values["Environment"] = event.environmentName || event.environmentId;
    }
    if (event?.releaseId) {
      values["Release"] = event.releaseName || event.releaseId;
    }
    if (event?.deploymentId) {
      values["Deployment ID"] = event.deploymentId;
    }
    if (event?.message) {
      values["Message"] = event.message;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnDeploymentEventConfiguration | undefined;
    const nodeMetadata = node.metadata as OctopusNodeMetadata | undefined;
    const metadata = buildMetadata(configuration, nodeMetadata);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: octopusIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      props.lastEventData = {
        subtitle: buildSubtitle(lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function buildMetadata(
  configuration: OnDeploymentEventConfiguration | undefined,
  nodeMetadata: OctopusNodeMetadata | undefined,
): TriggerProps["metadata"] {
  const metadata: TriggerProps["metadata"] = [];

  if (configuration?.eventCategories && configuration.eventCategories.length > 0) {
    const eventTypes = configuration.eventCategories.map((event: string) => formatEventLabel(event));
    metadata.push({
      icon: "funnel",
      label: eventTypes.length > 3 ? `Events: ${eventTypes.length} selected` : `Events: ${eventTypes.join(", ")}`,
    });
  }

  if (configuration?.project) {
    metadata.push({
      icon: "folder",
      label: `Project: ${nodeMetadata?.projectName || configuration.project}`,
    });
  }

  if (configuration?.environment) {
    metadata.push({
      icon: "globe",
      label: `Environment: ${nodeMetadata?.environmentName || configuration.environment}`,
    });
  }

  return metadata;
}

function buildSubtitle(createdAt?: string): string | React.ReactNode {
  return createdAt ? renderTimeAgo(new Date(createdAt)) : "";
}

/** Shared: value or "-" for display. */
export function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return String(value);
}

/** Shared: format timestamp for display, or "-" if missing/invalid. */
export function formatTimestamp(value?: string, fallback?: string): string {
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
