import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";
import githubIcon from '@/assets/icons/integrations/github.svg';
import { TriggerProps } from "@/ui/trigger";

interface GitHubMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  }
}

interface GithubConfiguration {
  events: string[];
}

interface GitHubEventData {
  head_commit?: {
    message?: string;
    id?: string;
  };
}

/**
 * Renderer for the "github" trigger type
 */
export const githubTriggerRenderer: TriggerRenderer = {
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as GitHubMetadata;

    let props: TriggerProps = {
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-black",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: [
        {
          icon: "book",
          label: metadata.repository.name,
        },
        {
          icon: "funnel",
          label: (node.configuration as unknown as GithubConfiguration).events.join(", "),
        }
      ],
      zeroStateText: "Waiting for the first push...",
    };

    if (lastEvent) {
      const eventData = lastEvent.data as GitHubEventData;
      props.lastEventData = {
        title: eventData?.head_commit?.message!,
        subtitle: eventData?.head_commit?.id,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "processed",
      };
    }

    return props;
  },
};
