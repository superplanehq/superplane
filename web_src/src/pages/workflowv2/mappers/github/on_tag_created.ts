import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, GitRef } from "./types";
import { Predicate, buildGithubSubtitle, createGithubMetadataItems } from "./utils";

interface GithubConfiguration {
  repository: string;
  tags: Predicate[];
}

/**
 * Renderer for the "github.onTagCreated" trigger
 */
export const onTagCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as GitRef;

    return {
      title: eventData?.ref ? `Tag: ${eventData.ref}` : "Tag Created",
      subtitle: buildGithubSubtitle(eventData?.ref || "", event.createdAt),
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as GitRef;

    return {
      Tag: eventData?.ref || "",
      Repository: eventData?.repository?.full_name || "",
      Sender: eventData?.sender?.login || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as GithubConfiguration;

    const props: TriggerProps = {
      title: node.name!,
      appName: "github",
      iconColor: getColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.tags),
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as GitRef;
      props.lastEventData = {
        title: eventData?.ref ? `Tag: ${eventData.ref}` : "Tag Created",
        subtitle: buildGithubSubtitle(eventData?.ref || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
