import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata } from "./base";

type PredicateType = "equals" | "notEquals" | "matches";

interface Predicate {
  type: PredicateType;
  value: string;
}

interface GithubConfiguration {
  repository: string;
  tags: Predicate[];
}

interface TagCreatedEventData {
  ref?: string;
  ref_type?: string;
  repository?: {
    name?: string;
    full_name?: string;
  };
  sender?: {
    login?: string;
  };
}

/**
 * Renderer for the "github.onTagCreated" trigger
 */
export const onTagCreatedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as TagCreatedEventData;

    return {
      title: eventData?.ref ? `Tag: ${eventData.ref}` : "Tag Created",
      subtitle: eventData?.repository?.full_name || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as TagCreatedEventData;

    return {
      Tag: eventData?.ref || "",
      Repository: eventData?.repository?.full_name || "",
      Sender: eventData?.sender?.login || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as GithubConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.tags) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.tags
          .map((tag) => {
            if (tag.type === "equals") {
              return `=${tag.value}`;
            }
            if (tag.type === "notEquals") {
              return `!=${tag.value}`;
            }
            if (tag.type === "matches") {
              return `~${tag.value}`;
            }
          })
          .join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as TagCreatedEventData;
      props.lastEventData = {
        title: eventData?.ref ? `Tag: ${eventData.ref}` : "Tag Created",
        subtitle: eventData?.repository?.full_name || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
