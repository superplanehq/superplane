import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";

interface GitHubMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

type PredicateType = "equals" | "notEquals" | "matches";

interface Predicate {
  type: PredicateType;
  value: string;
}

interface GithubConfiguration {
  refs: Predicate[];
}

interface PushEventData {
  head_commit?: {
    message?: string;
    id?: string;
    author?: {
      name?: string;
      email?: string;
      username: string;
    };
  };
}

/**
 * Renderer for the "github.onPush" trigger
 */
export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as PushEventData;

    return {
      title: eventData?.head_commit?.message || "",
      subtitle: eventData?.head_commit?.id || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data as PushEventData;

    return {
      Commit: eventData?.head_commit?.message || "",
      SHA: eventData?.head_commit?.id || "",
      Author: eventData?.head_commit?.author?.name || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as GitHubMetadata;
    const configuration = node.configuration as unknown as GithubConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.refs) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.refs
          .map((ref) => {
            if (ref.type === "equals") {
              return `=${ref.value}`;
            }
            if (ref.type === "notEquals") {
              return `!=${ref.value}`;
            }
            if (ref.type === "matches") {
              return `~${ref.value}`;
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
      const eventData = lastEvent.data as PushEventData;
      props.lastEventData = {
        title: eventData?.head_commit?.message || "",
        subtitle: eventData?.head_commit?.id || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
