import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import renderIcon from "@/assets/icons/integrations/render.svg";

interface RenderEventData {
  id?: string;
  serviceId?: string;
  serviceName?: string;
  status?: string;
}

interface OnEventConfiguration {
  eventTypes?: string[];
  serviceId?: string;
}

const eventLabels: Record<string, string> = {
  "render.deploy.ended": "Deploy Ended",
  "render.deploy.started": "Deploy Started",
  "render.build.ended": "Build Ended",
  "render.build.started": "Build Started",
  "render.image.pull.failed": "Image Pull Failed",
  "render.pipeline.minutes.exhausted": "Pipeline Minutes Exhausted",
  "render.pre.deploy.ended": "Pre-Deploy Ended",
  "render.pre.deploy.started": "Pre-Deploy Started",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Render Event";
  }

  return eventLabels[event] || event;
}

export const renderTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const event = context.event as RenderEventData | undefined;
    const title = buildTitle(event, context.event?.type as string);

    return {
      title,
      subtitle: buildSubtitle(context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const event = context.event?.data as RenderEventData | undefined;
    return {
      "Received At": toLocaleStringOrDash(context.event?.createdAt),
      Event: stringOrDash(context.event?.type),
      "Event ID": stringOrDash(event?.id),
      "Service ID": stringOrDash(event?.serviceId),
      "Service Name": stringOrDash(event?.serviceName),
      Status: stringOrDash(event?.status),
    };
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
    const eventTypes = configuration.eventTypes.map((event) => formatEventLabel(event));
    metadata.push({
      icon: "funnel",
      label: eventTypes.length > 3 ? `Events: ${eventTypes.length} selected` : `Events: ${eventTypes.join(", ")}`,
    });
  }

  if (configuration?.serviceId) {
    metadata.push({
      icon: "server",
      label: `Service: ${configuration.serviceId}`,
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

function stringOrDash(value?: string): string {
  if (!value) {
    return "-";
  }

  return value;
}

function toLocaleStringOrDash(value?: string, fallback?: string): string {
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
