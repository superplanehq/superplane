import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { formatPredicate, Predicate } from "../utils";
import renderIcon from "@/assets/icons/integrations/render.svg";

interface RenderEventData {
  id?: string;
  serviceId?: string;
  serviceName?: string;
  status?: string;
}

interface OnEventConfiguration {
  eventTypes?: string[];
  serviceIdFilter?: Predicate[];
  serviceNameFilter?: Predicate[];
}

const eventLabels: Record<string, string> = {
  "render.deploy.ended": "Deploy Ended",
  "render.deploy.started": "Deploy Started",
  "render.build.ended": "Build Ended",
  "render.build.started": "Build Started",
  "render.server.failed": "Server Failed",
  "render.server.available": "Server Available",
  "render.service.suspended": "Service Suspended",
  "render.service.resumed": "Service Resumed",
  "render.cron_job_run.ended": "Cron Job Run Ended",
  "render.job_run.ended": "Job Run Ended",
  "render.autoscaling.ended": "Autoscaling Ended",
  "render.deployment.failed": "Deployment Failed",
  "render.deployment.started": "Deployment Started",
  "render.deployment.succeeded": "Deployment Succeeded",
  "render.instance.deactivated": "Instance Deactivated",
  "render.instance.healthy": "Instance Healthy",
  "render.instance.unhealthy": "Instance Unhealthy",
  "render.service.deactivated": "Service Deactivated",
  "render.service.deploy.failed": "Service Deploy Failed",
  "render.service.deploy.started": "Service Deploy Started",
  "render.service.deploy.succeeded": "Service Deploy Succeeded",
  "render.service.live": "Service Live",
  "render.service.pre_deploy.failed": "Service Pre Deploy Failed",
  "render.service.restarted": "Service Restarted",
  "render.service.updated": "Service Updated",
  "render.service.update.failed": "Service Update Failed",
  "render.service.update.started": "Service Update Started",
};

function formatEventLabel(event?: string): string {
  if (!event) {
    return "Render Event";
  }

  return eventLabels[event] || event;
}

export const onEventTriggerRenderer: TriggerRenderer = {
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

  if (configuration?.serviceIdFilter && configuration.serviceIdFilter.length > 0) {
    metadata.push({
      icon: "server",
      label: formatFilterLabel("Service ID", configuration.serviceIdFilter),
    });
  }

  if (configuration?.serviceNameFilter && configuration.serviceNameFilter.length > 0) {
    metadata.push({
      icon: "tag",
      label: formatFilterLabel("Service Name", configuration.serviceNameFilter),
    });
  }

  return metadata;
}

function formatFilterLabel(name: string, predicates: Predicate[]): string {
  const formatted = predicates.map(formatPredicate);
  if (formatted.length > 2) {
    return `${name}: ${formatted.length} filters`;
  }

  return `${name}: ${formatted.join(", ")}`;
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
