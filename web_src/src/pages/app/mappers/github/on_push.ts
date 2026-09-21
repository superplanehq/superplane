import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata, Push } from "./types";
import { buildGithubSubtitle, createGithubMetadataItems } from "./utils";
import type { Predicate } from "../utils";

interface GithubConfiguration {
  refs: Predicate[];
  paths?: string[];
}

function pushShortSha(eventData?: Push): string {
  return eventData?.head_commit?.id?.slice(0, 7) || "";
}

function pushCommitMessage(eventData?: Push): string {
  return eventData?.head_commit?.message || "";
}

function pushRootEventValues(eventData?: Push): Record<string, string> {
  const headCommit = eventData?.head_commit;
  return {
    Commit: headCommit?.message || "",
    SHA: headCommit?.id || "",
    Author: headCommit?.author?.name || "",
  };
}

/**
 * Renderer for the "github.onPush" trigger
 */
export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as Push;

    return {
      title: pushCommitMessage(eventData),
      subtitle: buildGithubSubtitle(pushShortSha(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return pushRootEventValues(context.event?.data as Push);
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
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.refs, configuration?.paths),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as Push;
      props.lastEventData = {
        title: pushCommitMessage(eventData),
        subtitle: buildGithubSubtitle(pushShortSha(eventData), lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
