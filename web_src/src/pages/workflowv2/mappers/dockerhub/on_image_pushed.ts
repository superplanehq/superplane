import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer, TriggerRendererContext, TriggerEventContext } from "../types";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { TriggerProps } from "@/ui/trigger";
import { OnImagePushedMetadata, OnImagePushedConfiguration, WebhookPayload } from "./types";
import { formatTimeAgo } from "@/utils/date";

/**
 * Renderer for the "dockerhub.onImagePushed" trigger
 */
export const onImagePushedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as WebhookPayload;
    const tag = eventData?.push_data?.tag || "latest";
    const repoName = eventData?.repository?.repo_name || "";

    return {
      title: `${repoName}:${tag}`,
      subtitle: eventData?.push_data?.pusher ? `by ${eventData.push_data.pusher}` : "",
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as WebhookPayload;

    return {
      Repository: eventData?.repository?.repo_name || "",
      Tag: eventData?.push_data?.tag || "",
      Pusher: eventData?.push_data?.pusher || "",
      Namespace: eventData?.repository?.namespace || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnImagePushedMetadata;
    const configuration = node.configuration as unknown as OnImagePushedConfiguration;
    const metadataItems = [];

    if (metadata?.repository || configuration?.repository) {
      metadataItems.push({
        icon: "box",
        label: metadata?.repository || configuration?.repository || "",
      });
    }

    if (configuration?.tagFilter) {
      metadataItems.push({
        icon: "tag",
        label: `Filter: ${configuration.tagFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Docker Hub Push",
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as WebhookPayload;
      const tag = eventData?.push_data?.tag || "latest";
      const repoName = eventData?.repository?.repo_name || "";

      props.lastEventData = {
        title: `${repoName}:${tag}`,
        subtitle: eventData?.push_data?.pusher
          ? `by ${eventData.push_data.pusher} ${formatTimeAgo(new Date(lastEvent.createdAt!))}`
          : formatTimeAgo(new Date(lastEvent.createdAt!)),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
