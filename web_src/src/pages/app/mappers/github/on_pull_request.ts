import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata, PullRequest } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnPullRequestConfiguration {
  actions: string[];
}

interface OnPullRequestEventData {
  action?: string;
  number?: number;
  pull_request?: PullRequest;
}

function pullRequestEventTitle(eventData?: OnPullRequestEventData): string {
  return `#${eventData?.number} - ${eventData?.pull_request?.title}`;
}

function buildOnPullRequestMetadataItems(metadata?: BaseNodeMetadata, configuration?: OnPullRequestConfiguration) {
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

  return metadataItems;
}

function pullRequestRootEventValues(eventData?: OnPullRequestEventData): Record<string, string> {
  const pullRequest = eventData?.pull_request;
  return {
    URL: pullRequest?._links?.html?.href || "",
    Title: pullRequest?.title || "",
    Action: eventData?.action || "",
    Author: pullRequest?.user?.login || "",
  };
}

/**
 * Renderer for the "github.onPullRequest" trigger
 */
export const onPullRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnPullRequestEventData;

    return {
      title: pullRequestEventTitle(eventData),
      subtitle: buildGithubSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    return pullRequestRootEventValues(context.event?.data as OnPullRequestEventData);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnPullRequestConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildOnPullRequestMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnPullRequestEventData;

      props.lastEventData = {
        title: pullRequestEventTitle(eventData),
        subtitle: buildGithubSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
