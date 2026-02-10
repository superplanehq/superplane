import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { DockerHubImagePushEvent, DockerHubTriggerConfiguration, DockerHubTriggerMetadata } from "./types";
import { buildRepositoryMetadataItems, getRepositoryLabel } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { formatPredicate, stringOrDash } from "../../utils";

/**
 * Renderer for the "dockerhub.onImagePush" trigger
 */
export const onImagePushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as DockerHubImagePushEvent;
    const repository = getRepositoryLabel(undefined, undefined, eventData?.repository?.repo_name);
    const tag = eventData?.push_data?.tag;

    const title = repository ? `${repository}${tag ? `:${tag}` : ""}` : "Docker Hub image push";
    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt || "")) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as DockerHubImagePushEvent;
    const repository = eventData?.repository;
    const pushData = eventData?.push_data;
    const pushedAt = pushData?.pushed_at ? new Date(pushData.pushed_at * 1000).toISOString() : undefined;

    const visibility =
      repository?.is_private === undefined ? "-" : repository.is_private ? "Private" : "Public";

    return {
      Repository: stringOrDash(getRepositoryLabel(undefined, undefined, repository?.repo_name)),
      Tag: stringOrDash(pushData?.tag),
      Pusher: stringOrDash(pushData?.pusher),
      "Pushed At": pushedAt ? formatTimestampInUserTimezone(pushedAt) : "-",
      "Repository URL": stringOrDash(repository?.repo_url),
      Visibility: visibility,
      Stars: stringOrDash(repository?.star_count),
      Pulls: stringOrDash(repository?.pull_count),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as DockerHubTriggerMetadata | undefined;
    const configuration = node.configuration as DockerHubTriggerConfiguration | undefined;
    const metadataItems = buildRepositoryMetadataItems(metadata, configuration);

    if (configuration?.tags?.length) {
      metadataItems.push({
        icon: "tag",
        label: configuration.tags.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onImagePushTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
