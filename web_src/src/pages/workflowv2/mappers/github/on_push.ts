import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, Push } from "./types";
import { Predicate, buildGithubSubtitle, createGithubMetadataItems } from "./utils";

interface GithubConfiguration {
  refs: Predicate[];
}

/**
 * Renderer for the "github.onPush" trigger
 */
export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as Push;
    const shortSha = eventData?.head_commit?.id?.slice(0, 7) || "";

    return {
      title: eventData?.head_commit?.message || "",
      subtitle: buildGithubSubtitle(shortSha, event.createdAt),
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as Push;

    return {
      Commit: eventData?.head_commit?.message || "",
      SHA: eventData?.head_commit?.id || "",
      Author: eventData?.head_commit?.author?.name || "",
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
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.refs),
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as Push;
      const shortSha = eventData?.head_commit?.id?.slice(0, 7) || "";
      props.lastEventData = {
        title: eventData?.head_commit?.message || "",
        subtitle: buildGithubSubtitle(shortSha, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
