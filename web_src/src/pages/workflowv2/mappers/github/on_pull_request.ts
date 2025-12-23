import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";

interface OnPullRequestMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

interface OnPullRequestConfiguration {
  actions: string[];
}

interface OnPullRequestEventData {
  action?: string;
  number?: number;
  pull_request?: {
    _links?: {
      html?: {
        href: string;
      };
    };
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
 * Renderer for the "github.onPullRequest" trigger
 */
export const onPullRequestTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnPullRequestEventData;

    return {
      title: `#${eventData?.number} - ${eventData?.pull_request?.title}`,
      subtitle: eventData?.action || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnPullRequestEventData;

    return {
      URL: eventData?.pull_request?._links?.html?.href || "",
      Title: eventData?.pull_request?.title || "",
      Action: eventData?.action || "",
      Author: eventData?.pull_request?.user?.login || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnPullRequestMetadata;
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
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnPullRequestEventData;

      props.lastEventData = {
        title: `#${eventData?.number} - ${eventData?.pull_request?.title}`,
        subtitle: eventData?.action || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
