import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, GitRef } from "./types";
import { buildGithubSubtitle, createGithubMetadataItems } from "./utils";
import { Predicate } from "../utils";

interface GithubConfiguration {
  repository: string;
  branches: Predicate[];
}

/**
 * Renderer for the "github.onBranchCreated" trigger
 */
export const onBranchCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event.data as GitRef;

    return {
      title: eventData?.ref ? `Branch: ${eventData.ref}` : "Branch Created",
      subtitle: buildGithubSubtitle(eventData?.ref || "", context.event.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event.data as GitRef;

    return {
      Branch: eventData?.ref || "",
      Repository: eventData?.repository?.full_name || "",
      Sender: eventData?.sender?.login || "",
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
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.branches),
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as GitRef;
      props.lastEventData = {
        title: eventData?.ref ? `Branch: ${eventData.ref}` : "Branch Created",
        subtitle: buildGithubSubtitle(eventData?.ref || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
