import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata, GitRef } from "./types";
import { Predicate, createGithubMetadataItems } from "./utils";

interface GithubConfiguration {
  repository: string;
  branches: Predicate[];
}

/**
 * Renderer for the "github.onBranchCreated" trigger
 */
export const onBranchCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as GitRef;

    return {
      title: eventData?.ref ? `Branch: ${eventData.ref}` : "Branch Created",
      subtitle: eventData?.repository?.full_name || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as GitRef;

    return {
      Branch: eventData?.ref || "",
      Repository: eventData?.repository?.full_name || "",
      Sender: eventData?.sender?.login || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as GithubConfiguration;

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: createGithubMetadataItems(metadata?.repository?.name, configuration?.branches),
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as GitRef;
      props.lastEventData = {
        title: eventData?.ref ? `Branch: ${eventData.ref}` : "Branch Created",
        subtitle: eventData?.repository?.full_name || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
