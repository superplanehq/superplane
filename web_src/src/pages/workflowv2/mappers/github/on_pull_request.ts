import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, PullRequest } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnPullRequestConfiguration {
  actions: string[];
}

interface OnPullRequestEventData {
  action?: string;
  number?: number;
  pull_request?: PullRequest;
}

/**
 * Renderer for the "github.onPullRequest" trigger
 */
export const onPullRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnPullRequestEventData;

    return {
      title: `#${eventData?.number} - ${eventData?.pull_request?.title}`,
      subtitle: buildGithubSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnPullRequestEventData;

    return {
      URL: eventData?.pull_request?._links?.html?.href || "",
      Title: eventData?.pull_request?.title || "",
      Action: eventData?.action || "",
      Author: eventData?.pull_request?.user?.login || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnPullRequestConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.actions) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnPullRequestEventData;

      props.lastEventData = {
        title: `#${eventData?.number} - ${eventData?.pull_request?.title}`,
        subtitle: buildGithubSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
