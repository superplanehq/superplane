import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import { getDetailsForIssue } from "./base";
import type { BaseNodeMetadata, Issue } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnIssueConfiguration {
  actions: string[];
}

interface OnIssueEventData {
  action?: string;
  issue?: Issue;
}

function issueEventTitle(eventData?: OnIssueEventData): string {
  return `#${eventData?.issue?.number} - ${eventData?.issue?.title}`;
}

function buildOnIssueMetadataItems(metadata?: BaseNodeMetadata, configuration?: OnIssueConfiguration) {
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

/**
 * Renderer for the "github.onIssue" trigger
 */
export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnIssueEventData;

    return {
      title: issueEventTitle(eventData),
      subtitle: buildGithubSubtitle(eventData?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.issue as Issue;
    return getDetailsForIssue(issue);
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnIssueConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildOnIssueMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;

      props.lastEventData = {
        title: issueEventTitle(eventData),
        subtitle: buildGithubSubtitle(eventData?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
