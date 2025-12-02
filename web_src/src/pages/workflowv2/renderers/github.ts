import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";

interface GitHubMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

interface RefFilter {
  type: string;
  value: string;
}

interface GithubConfiguration {
  eventType: string;
  refs?: RefFilter[];
}

interface GitHubEventData {
  head_commit?: {
    message?: string;
    id?: string;
    author?: {
      name?: string;
      email?: string;
      username: string;
    };
  };
  pull_request?: {
    title?: string;
    id?: string;
    url?: string;
    head?: {
      sha: string;
      ref: string;
    };
    user?: {
      id: string;
      login: string;
    };
  };
}

/**
 * Renderer for the "github" trigger type
 */
export const githubTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as GitHubEventData;

    if (eventData.pull_request) {
      return {
        title: eventData?.pull_request?.title || "",
        subtitle: eventData?.pull_request?.head?.sha || "",
      };
    }

    return {
      title: eventData?.head_commit?.message || "",
      subtitle: eventData?.head_commit?.id || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data as GitHubEventData;

    if (eventData.pull_request) {
      return {
        Commit: eventData?.pull_request?.title || "",
        SHA: eventData?.pull_request?.head?.sha || "",
        Author: eventData?.pull_request?.user?.login || "",
      };
    }

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

    // Only add repository metadata if it exists
    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    // Only add events metadata if configuration exists
    if (configuration?.eventType) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.eventType,
      });
    }

    // Add ref filters if they exist
    if (configuration?.refs && configuration.refs.length > 0) {
      const branchLabels = configuration.refs.map((ref) => {
        return ref.value;
      });

      metadataItems.push({
        icon: "git-branch",
        label: branchLabels.join(", "),
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
      zeroStateText: "Waiting for the first push...",
    };

    if (lastEvent) {
      const eventData = lastEvent.data as GitHubEventData;

      if (eventData.pull_request) {
        props.lastEventData = {
          title: eventData?.pull_request?.title || "",
          subtitle: eventData?.pull_request?.head?.sha || "",
          receivedAt: new Date(lastEvent.createdAt!),
          state: "processed",
        };
      } else {
        props.lastEventData = {
          title: eventData?.head_commit?.message || "",
          subtitle: eventData?.head_commit?.id || "",
          receivedAt: new Date(lastEvent.createdAt!),
          state: "processed",
        };
      }
    }

    return props;
  },
};
