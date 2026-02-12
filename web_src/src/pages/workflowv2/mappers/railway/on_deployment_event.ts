import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import RailwayLogo from "@/assets/icons/integrations/railway.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnDeploymentEventMetadata {
  project?: {
    id: string;
    name: string;
  };
}

interface OnDeploymentEventConfiguration {
  statuses?: string[];
}

interface DeploymentResource {
  owner?: { id?: string; email?: string };
  project?: { id?: string; name?: string };
  environment?: { id?: string; name?: string };
  service?: { id?: string; name?: string };
  deployment?: { id?: string };
}

interface OnDeploymentEventData {
  type?: string;
  details?: { status?: string };
  resource?: DeploymentResource;
  timestamp?: string;
}

/**
 * Renderer for the "railway.onDeploymentEvent" trigger type
 */
export const onDeploymentEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnDeploymentEventData;
    const serviceName = eventData?.resource?.service?.name || "Service";
    const status = eventData?.details?.status || "";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = status && timeAgo ? `${status.toLowerCase()} · ${timeAgo}` : status.toLowerCase() || timeAgo;

    return {
      title: `${serviceName} deployment`,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnDeploymentEventData;
    const resource = eventData?.resource;
    const receivedAt = context.event?.createdAt
      ? new Date(context.event?.createdAt).toLocaleString()
      : "";

    // Build Railway deployment URL
    const projectId = resource?.project?.id;
    const serviceId = resource?.service?.id;
    const deploymentId = resource?.deployment?.id;
    const deploymentLink =
      projectId && serviceId
        ? `https://railway.com/project/${projectId}/service/${serviceId}${deploymentId ? `?deploymentId=${deploymentId}` : ""}`
        : "";

    return {
      "Received at": receivedAt,
      Status: eventData?.details?.status || "",
      Service: resource?.service?.name || "",
      Environment: resource?.environment?.name || "",
      Project: resource?.project?.name || "",
      "Deployment link": deploymentLink,
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnDeploymentEventMetadata;
    const configuration = node.configuration as unknown as OnDeploymentEventConfiguration;
    const metadataItems = [];

    // Show project name
    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "folder",
        label: metadata.project.name,
      });
    }

    // Show status filter if configured
    if (configuration?.statuses && configuration.statuses.length > 0) {
      metadataItems.push({
        icon: "filter",
        label: configuration.statuses.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "On Deployment Event",
      iconSrc: RailwayLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnDeploymentEventData;
      const serviceName = eventData?.resource?.service?.name || "Service";
      const status = eventData?.details?.status || "";
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = status && timeAgo ? `${status.toLowerCase()} · ${timeAgo}` : status.toLowerCase() || timeAgo;

      props.lastEventData = {
        title: `${serviceName} deployment`,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
