import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Push } from "./types";
import { buildGithubSubtitle, createGithubMetadataItems } from "./utils";
import { Predicate } from "../utils";

interface GithubConfiguration {
  refs: Predicate[];
}

/**
 * Renderer for the "github.onPush" trigger
 */
export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as Push;
    const shortSha = eventData?.head_commit?.id?.slice(0, 7) || "";

    return {
      title: eventData?.head_commit?.message || "",
      subtitle: buildGithubSubtitle(shortSha, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as Push;

    return {
      Commit: eventData?.head_commit?.message || "",
      SHA: eventData?.head_commit?.id || "",
      Author: eventData?.head_commit?.author?.name || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as GithubConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.refs),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as Push;
      const shortSha = eventData?.head_commit?.id?.slice(0, 7) || "";
      props.lastEventData = {
        title: eventData?.head_commit?.message || "",
        subtitle: buildGithubSubtitle(shortSha, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
