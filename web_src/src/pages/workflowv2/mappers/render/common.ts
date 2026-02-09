import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import renderIcon from "@/assets/icons/integrations/render.svg";

interface RenderEventData {
  eventId?: string;
  serviceId?: string;
  serviceName?: string;
  status?: string;
  deployId?: string;
  buildId?: string;
}

interface OnEventConfiguration {
  eventTypes?: string[];
  service?: string;
}

/** Labels for event types as received in payloads (dot-case, e.g. render.deploy.ended). */
const eventLabelsByType: Record<string, string> = {
  "render.deploy.ended": "Deploy Ended",
  "render.deploy.started": "Deploy Started",
  "render.build.ended": "Build Ended",
  "render.build.started": "Build Started",
  "render.image.pull.failed": "Image Pull Failed",
  "render.pipeline.minutes.exhausted": "Pipeline Minutes Exhausted",
  "render.pre.deploy.ended": "Pre-Deploy Ended",
  "render.pre.deploy.started": "Pre-Deploy Started",
};

/** Labels for event types as stored in configuration (snake_case, e.g. deploy_ended). */
const eventLabelsByConfig: Record<string, string> = {
  deploy_ended: "Deploy Ended",
  deploy_started: "Deploy Started",
  build_ended: "Build Ended",
  build_started: "Build Started",
  image_pull_failed: "Image Pull Failed",
  pipeline_minutes_exhausted: "Pipeline Minutes Exhausted",
  pre_deploy_ended: "Pre-Deploy Ended",
  pre_deploy_started: "Pre-Deploy Started",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Render Event";
  }

  return eventLabelsByType[event] || eventLabelsByConfig[event] || event;
}

export const renderTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const event = context.event?.data as RenderEventData | undefined;
    const title = buildTitle(event, context.event?.type as string);

    return {
      title,
      subtitle: buildSubtitle(context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = context.event?.data as RenderEventData | undefined;
    const values: Record<string, string> = {
      "Received At": formatTimestamp(context.event?.createdAt),
      Event: stringOrDash(context.event?.type),
      "Event ID": stringOrDash(event?.eventId),
      "Service ID": stringOrDash(event?.serviceId),
      "Service Name": stringOrDash(event?.serviceName),
    };

    if (event?.deployId) {
      values["Deploy ID"] = event.deployId;
    }
    if (event?.buildId) {
      values["Build ID"] = event.buildId;
    }

    if (event?.status) {
      values["Status"] = event.status;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnEventConfiguration | undefined;
    const metadata = buildMetadata(configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: renderIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata,
    };

    if (lastEvent) {
      const event = lastEvent.data as RenderEventData;
      props.lastEventData = {
        title: buildTitle(event, lastEvent.type as string),
        subtitle: buildSubtitle(lastEvent.createdAt),
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

  if (configuration?.eventTypes && configuration.eventTypes.length > 0) {
    const eventTypes = configuration.eventTypes.map((event: string) => formatEventLabel(event));
    metadata.push({
      icon: "funnel",
      label: eventTypes.length > 3 ? `Events: ${eventTypes.length} selected` : `Events: ${eventTypes.join(", ")}`,
    });
  }

  const service = configuration?.service;
  if (service) {
    metadata.push({
      icon: "server",
      label: `Service: ${service}`,
    });
  }

  return metadata;
}

function buildTitle(event: RenderEventData | undefined, type?: string): string {
  const serviceLabel = event?.serviceName || event?.serviceId || "Service";
  const eventLabel = formatEventLabel(type);
  return `${serviceLabel} · ${eventLabel}`;
}

function buildSubtitle(createdAt?: string): string {
  return createdAt ? formatTimeAgo(new Date(createdAt)) : "";
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
